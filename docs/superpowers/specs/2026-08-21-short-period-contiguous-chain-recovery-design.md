# 短周期正式方案连续期链路修复设计

## 背景与已确认根因

正式方案 `def-1010-1787289800308` 对应实例 `inst-1010-1787289800379` 已成功完成首期 `10114256504789` 下单，第三方注单号为 `128601900`。该期开奖结果在开奖后约 945ms 写入 `lottery_draws`，但策略评估、倍数轮次推进和下一期 Outbox 均未生成。

现场数据与代码链路共同证明：

1. 第三方开奖 WebSocket 的彩种边界状态落后 REST 当前期约 17 期，现有连接没有读超时、ping/pong 或按彩种静默检测，半开连接可能永久阻塞在 `ReadMessage`。
2. 3秒、6秒、15秒正式方案要求 WebSocket 的 `CurrentIssue` 必须等于待评估来源期，且 `NextIssue` 必须是目标期；该约束用于防止第三方实际接收成其他期。
3. 当前策略评估、实例状态推进和下一期指令创建处于同一事务。获取不到新鲜连续目标期时整个事务回滚，已存在的开奖结果也无法被标记为已评估。
4. 如果 REST 先写入开奖，随后到达的 WebSocket 同期开奖会命中去重；现有通知只在首次插入开奖时触发，新的 WebSocket 期号边界不会再次唤醒等待目标期的决策。
5. 启动接管会尝试重新武装所有 `blocked_requires_rearm` 实例。连续期已经错过的实例不能进入该自动恢复范围，否则会跨过缺失期建立新指令。

问题不是 NATS、分片或 Outbox 总体架构失效，而是 WebSocket 生存性、策略事务边界和连续期失败语义缺失。修复保留现有事件驱动、分片、租约、Outbox 和幂等派单架构。

## 目标

- 正常情况下，正式短周期方案仅根据已被第三方接受的上一期 `N` 的开奖号码推进策略，并只向紧邻下一期 `N+1` 下单。
- 开奖事实入库后立即持久化本地中奖判定、倍数、轮次和选号状态，不受第三方财务结算延迟影响。
- WebSocket 短暂抖动时自动恢复；不增加按用户或按方案访问第三方的轮询。
- 已错过紧邻下一期时停止该实例，记录 `missed_contiguous_period`，不得自动跳到当前期继续。
- 后端重启、NATS 重投、重复开奖、接口超时和并发 worker 不得重复推进状态或重复下注。
- 一个彩种或实例的异常不得阻塞其他彩种、用户、实例或分片。
- 保持面向日均至少 500 万注的按彩种汇聚、按实例分片和有界并发模型。

## 非目标

- 不恢复已经过去的物理期，也不伪造历史下注。
- 不使用 REST periods 快照直接授权 3秒、6秒、15秒正式下单；该接口可用于诊断和检测 WebSocket 落后，但不能重新引入 `target=N, accepted=N+1`。
- 不改变玩法规则、金额、倍数、单挑、第三方 payload 或财务结算口径。
- 不推翻 JetStream、PostgreSQL Outbox、分片租约或第三方精确指纹对账链路。

## 核心不变量

每一笔连续期正式下注必须同时满足：

1. 来源注单已被第三方明确接受，且本地保存了不可变玩法规则快照。
2. `source_period` 等于该实例最后一笔已接受且尚未推进策略的期号。
3. 本地开奖事实的 `issue_no` 等于 `source_period`。
4. WebSocket 边界的 `CurrentIssue` 等于 `source_period`。
5. WebSocket 边界的 `NextIssue` 是仍处于安全下单窗口的唯一目标期。
6. 第三方返回的实际接受期等于冻结请求中的目标期。
7. `(scheme_id, source_period)` 最多推进一次策略，`(decision_id)` 最多生成一个 Outbox，`request_id` 最多产生一次外部投注。

若第 1 至 3 项不满足，等待开奖或规则条件；若第 4、5 项在紧邻期安全截止前仍不满足，则连续链失败；若第 6 项不满足，立即阻断为第三方接收错期。

## 方案选择

### 方案 A：持续回滚并等待 WebSocket 恢复

保持现有单事务，只增加 WebSocket 重连。实现改动较小，但 WebSocket 恢复时通常已进入后续期，旧来源期永远无法再次满足严格匹配，实例仍会永久显示运行中。否决。

### 方案 B：恢复 REST 目标期兜底

WebSocket 不可用时直接使用 periods REST 当前期。该方案能提高表面下单率，但已经在现场出现第三方接受为下一期的问题，违反不跨期要求。否决。

### 方案 C：持久化策略结果，独立等待连续目标期（采用）

将策略判定和目标期派单拆成两个幂等阶段。开奖结果一旦有效即完成策略推进并写入一个 `awaiting_target` 决策；WebSocket 边界在安全窗口内到达时为该决策创建唯一 Outbox。安全截止后仍不能证明连续目标期，则把决策和实例标记为 `missed_contiguous_period`，停止链路。

该方案保留严格期号安全，又消除“拿不到目标期导致已完成判定反复回滚”的永久阻塞。

## 数据模型

### 方案决策

扩展 `scheme_period_decisions.status`：

- `awaiting_target`：策略结果和实例状态已持久化，正在等待来源期对应的 WebSocket 下一期边界。
- `completed`：唯一 Outbox 已创建，决策完成。
- `missed_contiguous_period`：安全截止前未取得合法连续目标期，禁止继续当前 chain。
- 保留现有 `blocked`、`duplicate`、`chain_broken` 兼容已有记录。

新增字段：

- `target_deadline_at TIMESTAMPTZ`：来源期开奖后，紧邻下一期允许创建指令的最后安全时刻。
- `target_period_no VARCHAR(64)`：WebSocket 证明连续目标期后写入；等待期间为空。
- `failure_reason VARCHAR(64)`：仅保存稳定机器码，例如 `draw_ws_stale`、`next_period_unavailable`、`missed_contiguous_period`。

`UNIQUE (scheme_id, source_period_no)` 保持不变，保证同一期最多一个策略决策。

### 实例连续链

在 `scheme_instances` 新增可空字段 `chain_block_reason VARCHAR(64)`：

- chain 正常激活时清空。
- 连续期错过时写入 `missed_contiguous_period`，同时设置 `strict_chain_state='blocked_requires_rearm'`、`status='paused'`、`status_reason='bet_failed'`。
- `bet_failed_detail` 保存来源期、观察到的 WebSocket/REST 期号和时间线，供管理员诊断；用户端继续使用既有通用失败展示。
- 自动 rearm 仅允许已有安全白名单原因，例如明确未出站的封盘前失败；`missed_contiguous_period`、接收错期和外部接收不明确不得自动 rearm。
- 用户手工重新开启时建立新 chain，清除 block reason，重置倍投轮次并从当时已证明可投的目标期开始。

## WebSocket 生存性设计

### 连接级健康

- 使用单个第三方开奖连接服务多个彩种，不为用户或方案新增连接。
- 设置 ping writer、pong handler 和读截止时间；连接未收到任何帧或 pong 超过连接级阈值时主动关闭，由现有退避循环重连。
- 重连退避设置上限并加入抖动，避免多节点同时重连；后端退出时立即取消 ping 和重连 goroutine。

### 彩种级健康

- 每次解析出有效 `DrawEvent` 后，按内部 lottery code 更新 `CurrentIssue`、`NextIssue`、接收时间和 interval。
- 每个短周期彩种根据上一条有效边界的本地单调接收时刻计算下一条预期到达点；宽限固定为 `min(500ms, interval/6)`。超过“上一条接收时刻 + interval + 宽限”仍无该彩种有效边界即判定 stale，不能再等待完整的下一周期。其他彩种仍有 WebSocket 消息不能掩盖该彩种的静默。
- 彩种静默时触发一次合并后的 WebSocket 重连请求，并记录 `draw_ws_stale`；同一连接的并发请求只执行一次重连。重连后若在 `target_deadline_at` 前补到来源期边界，仍可生成紧邻下一期指令；否则进入连续期中断。
- REST history/periods 继续运行，用于补开奖事实和比较落后期数，但不授权正式短周期下单。
- 管理员诊断展示最后有效帧、最后 pong、最后有效彩种边界、重连次数、连续失败次数以及 WS 相对 REST 的落后期数。

## 两阶段策略与派单

### 阶段一：开奖判定事务

开奖事件或数据库恢复扫描找到已接受注单 `N` 后：

1. 锁定实例状态版本并声明 `(instance_id, N)` 策略评估。
2. 使用注单冻结的规则和投注内容计算本地命中结果。
3. 推进 `round_index`、`pick_index`、`current_pick` 和 `last_direction`，状态版本加一。
4. 完成 `scheme_strategy_evaluations`，写入 `cloud_bet_records.strategy_evaluated_at`。
5. 创建唯一 `scheme_period_decisions`，状态为 `awaiting_target`。`target_deadline_at` 使用数据库时钟和来源期开奖时间计算为“紧邻下一期关闭时间减现有出站安全余量”；开奖到达时已超过该时刻则直接进入连续期中断。
6. 提交事务。此阶段不要求下一期目标已经可用，也不创建第三方请求。

重复开奖、NATS 重投和恢复扫描命中唯一键后直接读取现有决策，不重复推进倍数或轮次。

### 阶段二：连续目标期解析

以下事件均可唤醒等待决策：

- 同一彩种新的 WebSocket 期号边界，即使开奖行此前已由 REST 插入、此次写入被去重。
- `awaiting_target` 的 JetStream 重投。
- 后端启动恢复。
- 按彩种、按小批量执行的数据库兜底恢复，不按方案请求第三方。

解析器仅在 `WS.CurrentIssue == decision.source_period_no`、`WS.NextIssue` 非空且安全窗口充足时：

1. 冻结使用阶段一更新后状态生成的投注请求。
2. 在一个事务中把决策更新为 `completed`，写入 `target_period_no`，创建唯一 Outbox 并递增 chain sequence。
3. 发布 `bet.ready`；发布失败由数据库 Outbox 恢复，不回滚已经提交的决策和指令。

如果 WS 已经明确前进到来源期之后，或 `target_deadline_at` 已过，则原子地把决策标记为 `missed_contiguous_period` 并阻断实例。不得使用更新后的状态为更晚期生成指令。

目标解析器与到期处理器可能并发访问同一决策。两者都必须锁定决策和实例，且仅允许从 `status='awaiting_target'` 条件更新：目标解析器还要求数据库时钟早于 `target_deadline_at`，到期处理器要求数据库时钟不早于该时刻。先成功提交的一方决定最终状态，另一方读到零更新后退出，禁止依赖进程时间先后猜测结果。

## 出站与对账

- 出站继续使用冻结请求、数据库租约、分片限流、安全截止时间和唯一 request ID。
- 第三方实际接受期与目标期不一致时，保留实际期号和响应摘要，设置 chain block reason 为 `provider_accepted_wrong_period`，不创建后续决策。
- 请求可能已写出但响应不明确时继续使用精确投注指纹对账；未证明未出站前不得重发。
- 第三方财务结算继续异步更新金额和盈亏，不覆盖已经由策略事务推进的倍数、轮次和选号状态。

## 重启与恢复

- 启动时先恢复 WebSocket、JetStream consumer 和分片租约，再扫描 `awaiting_target` 决策。
- 决策仍在安全窗口且 WebSocket 边界匹配时创建 Outbox；窗口已过时标记连续期中断。
- 已存在 Outbox 的决策不得再次创建；已接受注单不得再次发送。
- `chain_block_reason='missed_contiguous_period'` 的实例不进入自动 rearm。
- 用户手工重新开启是新 chain，不复用旧 `chain_id`、`chain_seq` 或等待决策，倍投轮次从第一轮开始。

## 并发与性能

- WebSocket、REST 快照和 stale 检测均按彩种共享，外部请求量不随用户或方案数量增长。
- 一次开奖或边界事件按 lottery code 分页展开相关候选，再按 scheme ID 分片；每页和每分片使用现有有界并发。
- 热路径只使用带索引的候选查询、实例行锁、唯一键和内存规则快照，不做全表扫描或逐方案规则 JOIN。
- `awaiting_target` 恢复索引使用 `(lottery_code, status, target_deadline_at, id)`，到期扫描按固定批量进行。
- 一个实例失败只更新该实例及其决策；一个分片处理失败通过 JetStream 重投或数据库恢复，不持有全局锁。
- 不增加逐方案定时器、逐方案 periods 请求或逐注单结算请求。

## 诊断语义

管理员诊断按实际阶段返回单一主阻塞原因：

- `draw_missing`：来源期尚无开奖事实。
- `strategy_evaluation_failed`：规则解析或本地判定失败。
- `draw_ws_stale`：开奖事实存在，但彩种 WebSocket 边界过期。
- `next_period_unavailable`：仍在安全窗口内，尚未取得连续目标期。
- `missed_contiguous_period`：安全窗口已过，当前 chain 已停止。
- `provider_accepted_wrong_period`：第三方实际接收期不同。
- `provider_acceptance_unknown`：请求可能出站但尚未完成精确对账。

每项诊断至少携带实例、chain、来源期、目标期、状态版本、最后 WS/REST 期号、关键时间点和 Outbox/第三方订单标识；日志不得包含 token、密码或完整凭证。

## 测试与验收

### 单元与集成测试

1. WebSocket 无 pong 或无任何帧时触发重连，取消上下文后 goroutine 全部退出。
2. 其他彩种仍有帧但 `tron_ffc_6s` 超过一个周期无有效边界时，按彩种判定 stale 并合并重连。
3. REST 先插入开奖、WS 后到同期开奖时，即使数据库插入被去重，仍唤醒 `awaiting_target` 决策。
4. 注单已接受且开奖存在时，策略评估和倍数轮次只推进一次；目标期暂缺不回滚评估。
5. WebSocket 在安全截止前提供 `N -> N+1` 时，只创建一个决策 Outbox，冻结请求使用推进后的倍数、轮次和选号。
6. WebSocket 已到 `N+1` 或安全截止已过时，实例变为 `missed_contiguous_period`，不为 `N+2` 创建订单。
7. NATS 重复消息、数据库恢复和后端重启不会重复推进或重复下单。
8. 第三方接受错误期、接收不明确和明确未出站分别进入不同状态，只有明确未出站的安全白名单允许自动 rearm。
9. 手工重新开启创建新 chain 并重置倍投轮次。
10. 财务结算先于或晚于本地策略评估时，最终金额、盈亏、倍数和轮次均保持各自权威来源。

### 现场验收

以新建或手工重启的 `tron_ffc_6s` 正式方案做灰度验证：

- 连续观察至少 100 个物理期。
- 每笔记录满足 `source N -> target N+1`，且本地目标期、第三方接受期一致。
- 每期只允许一笔该实例订单，倍数和轮次与上一期本地判定一致。
- 正常 WebSocket 条件下，开奖入库、策略评估和下一期 Outbox 创建总延迟目标不超过 3 秒。
- 人工断开 WebSocket 后，连接自动重建；若已错过紧邻期，实例明确停止且不向更晚期补投。
- 后端重启后，安全窗口内的等待决策自动恢复；已错期实例不自动 rearm。

### 性能验收

- 用批量候选和重复事件压测验证每个实例每期最多一个决策和一个 Outbox。
- 验证外部 WebSocket/periods/history 请求量只和彩种、账户有关，不随运行方案数线性增长。
- 验证单彩种积压或规则失败不阻塞其他分片和彩种。
- 保留数据库慢查询、JetStream pending/ack pending、策略处理延迟和 Outbox 派单延迟指标，作为五百万级日注量上线基线。

## 发布顺序与回滚

1. 先应用数据库迁移，再部署支持新决策状态和 chain block reason 的后端。
2. 以 `SCHEME_BETTING_MODE=gray` 和单个短周期彩种灰度，不同时扩大彩种范围。
3. 验证正常连续链、WebSocket 人工中断、后端重启和第三方错期保护后再扩大。
4. 若新策略阶段异常，停止新增正式 chain，保留已接受注单和诊断数据；不得回滚数据库并让旧代码解释新状态。
5. 回滚应用版本前必须先将所有 `awaiting_target` 决策收敛为 `completed` 或 `missed_contiguous_period`，避免旧代码跳过未知状态。

## 与既有设计的关系

本设计补充并收紧以下既有文档，不改变其事件驱动方向：

- `2026-08-17-ws-strategy-settlement-design.md`：落实“本地策略与第三方财务结算解耦”。
- `2026-08-18-short-period-continuity-and-settlement-refresh-design.md`：保留按彩种汇聚和有界并发。
- `2026-08-18-short-period-phase-lock-fix-design.md`：保留出站前双快照和安全窗口校验。
- `2026-08-20-shared-provider-period-snapshot.md`：保留按彩种共享快照，禁止恢复逐方案同步请求。

新增约束是：短周期 WebSocket 必须具备生存性监督；策略结果必须先持久化；错过紧邻期必须停止当前 chain，且不得由启动接管或自动 rearm 跨期恢复。
