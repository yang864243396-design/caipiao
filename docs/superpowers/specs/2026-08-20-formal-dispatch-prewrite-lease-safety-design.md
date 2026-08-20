# 正式派发请求写出边界与租约安全设计

日期：2026-08-20

## 背景与已确认事实

方案定义 `def-1-1787058821158` 对应实例 `inst-1-1787058821216`，彩种为 `tron_ffc_6s`，第三方 `game_id=74`。第三方最后一笔真实接单停在 `10114251003180`（约 09:45）；此后事件驱动正式链路创建的 outbox 均未形成第三方投注记录。

outbox `606` 的目标期 `10114251900908` 来自当时第三方 periods 快照。`101142519` 号段在 13:57:26 至 14:27:31 有效，因此本次修复不修改彩种映射或期号号段算法。

outbox `606` 的请求尝试持续约 10 秒，随后本地进入 `external_acceptance_unknown`，实例进入 `blocked_requires_rearm`。同机诊断新建 HTTP 客户端访问第三方投注历史时，也可稳定复现约 10 秒的 `TLS handshake timeout`。后端主机时间比 PostgreSQL 慢约 0.97 秒；现有 outbox lease 使用应用时钟创建和续期，存在短周期彩种下的租约误失效风险。

## 目标

1. 精确区分“HTTP 请求尚未写出”和“请求可能已写出”。
2. DNS、TCP、代理连接和 TLS 握手阶段失败时，明确按未发送失败处理，不阻塞整个方案链。
3. 请求体可能已写出后发生超时或断连时，继续保守进入未知接单状态，绝不自动重投。
4. outbox lease 的创建、判断和续期统一使用 PostgreSQL 时钟，消除应用主机时钟偏差。
5. 单个方案或单个彩种异常不得阻塞其他用户、彩种或 dispatcher 分片。
6. 为开发者保存足以定位请求阶段、租约阶段和耗时的诊断证据。

## 非目标

- 不改变投注内容、注数、金额、倍数、单挑或玩法规则。
- 不改变 `tron_ffc_6s` 的 `game_id=74` 映射。
- 不根据本地推断自动确认未知第三方订单。
- 不对请求体可能已写出的下注执行自动重试。
- 不在本次修改中重新设计 periods、开奖或策略计算流程。

## 方案选择

### 方案 A：按错误字符串判断 TLS 超时

仅将包含 `TLS handshake timeout` 的错误标记为未发送。改动小，但依赖错误文本，无法可靠覆盖代理、连接重置和不同 Go/Windows 网络栈错误，不采用。

### 方案 B：请求写出状态追踪 + 数据库时钟租约（采用）

在第三方 HTTP 边界使用 `httptrace.ClientTrace.WroteRequest` 记录请求是否已经交给传输层写出；仅当回调中的 `WroteRequestInfo.Err == nil` 时，才将 `RequestWritten` 置为 true。错误返回时携带结构化的 `RequestWritten`、阶段和耗时。outbox lease 在 SQL 内使用 `clock_timestamp()` 创建、判断和续期。该方案能同时解决重复下注安全和租约时钟偏差，且不增加第三方调用次数。

### 方案 C：暂时回退到 legacy worker

可较快恢复旧下注，但会绕开已设计的严格链、事件总线和前一期结果栅栏，重新引入跨期、并发重复和状态不一致风险，不采用。

## HTTP 请求写出边界

### 结构化传输错误

第三方 HTTP 客户端返回可解包的结构化错误，至少包含：

- `Operation`：请求方法与路径，例如 `POST /api/web_bets/lott`。
- `Phase`：`dns`、`connect`、`tls`、`write`、`response` 或 `decode`。
- `RequestWritten`：是否收到 `httptrace.WroteRequest` 回调。
- `Duration`：从开始请求到失败的总耗时。
- `Cause`：原始错误，保留 `errors.Is`/`errors.As` 能力。

`WroteRequest` 表示 HTTP 请求已经被传输层写出，不表示第三方业务接单成功。它仅作为“是否允许确定未发送”的安全边界。

### 错误分类

| 条件 | outbox 结果 | 是否阻塞严格链 | 是否允许自动进入下一期 |
|---|---|---:|---:|
| `RequestWritten=false` | `rejected`，reason=`provider_pre_send_failed` | 否 | 是 |
| `RequestWritten=true` 且无明确业务拒绝 | `sent_unknown`，到截止时间转 `external_acceptance_unknown` | 是 | 否 |
| 第三方明确返回余额不足、封盘、参数错误等业务拒绝 | `rejected` | 按现有业务规则 | 按现有业务规则 |
| 返回订单号且期号一致 | `accepted` | 否 | 是 |
| 返回订单号但期号不一致 | `accepted_wrong_period` | 是 | 否 |

DNS、TCP、TLS 握手通常发生在请求写出之前，但最终分类只依赖 `RequestWritten`，不依赖错误字符串。

`provider_pre_send_failed` 不在同一个 outbox 内重试。策略层可以在下一开放期基于未变化的方案状态创建一个新的唯一 outbox；这不是推进输赢轮次，也不会伪造上一期结果。

### 请求截止时间

正式下注请求使用 outbox 的 `safe_deadline_at` 作为最晚截止时间，并取其与全局 HTTP timeout 的较早值。这样连接失败能在封盘安全线前结束，不会由 30 秒全局 timeout 占满短周期彩种窗口。

请求 context 到期后：

- 未收到 `WroteRequest`：确定未发送。
- 已收到 `WroteRequest`：未知接单，不重试。

## PostgreSQL 时钟租约

### StartAttempt

`StartAttempt` 在同一条 SQL 中完成：

1. 校验 outbox 为当前 owner 和 fencing token 持有的 `leased` 状态。
2. 使用 `clock_timestamp()` 判断现有 lease 是否仍有效。
3. 写入 `dispatch_started_at=clock_timestamp()`。
4. 将 `lease_until` 重置为 `clock_timestamp() + lease_duration`。
5. 创建 attempt 记录。

不得再使用 run loop 开始时缓存的应用 `now` 作为真实下单租约的最终依据。

### Heartbeat

Heartbeat SQL 使用数据库当前时间判断并续期：

```text
lease_until > clock_timestamp()
lease_until = clock_timestamp() + lease_duration
```

应用只传递租约时长，不传递绝对 `lease_until`。这样后端、数据库和其他节点存在时钟偏差时，租约仍以单一时钟源运行。

### 心跳失败行为

- 心跳 SQL 报错或 fencing 失效时，记录结构化诊断，包括 owner、token、上次成功续期时间和错误。
- 心跳失败不会自动重试第三方请求。
- 如果请求尚未写出，取消请求并按 `provider_pre_send_failed` 完成。
- 如果请求已经写出，取消只能缩短等待，最终仍按未知接单处理。
- sweeper 继续负责进程崩溃后的恢复，但不能覆盖已经保存的原始传输错误。

## 并发与性能

- `httptrace` 是单请求内存状态，不增加外部请求或数据库查询。
- Heartbeat 仍按现有租约周期运行，不增加频率；SQL 只更新当前 outbox 主键行。
- periods 继续按彩种共享快照和 singleflight 刷新，不恢复为每方案刷新。
- 各 outbox 使用独立 fencing token；单个请求卡住仅占用一个 dispatcher concurrency 槽位。
- dispatcher 的彩种、账户和全局限流保持不变。
- 不新增轮询，也不新增数据库表或索引。

## 诊断与审计

`scheme_bet_outbox.last_error` 和最新 `scheme_bet_attempts.error_message` 保存以下信息：

```text
provider placement failed phase=tls request_written=false verify_ms=... place_ms=... cause=...
```

租约失败保存：

```text
lease heartbeat failed owner=... token=... last_success_at=... cause=...
```

reconciliation 只能在原始错误为空时写入兜底错误，不能覆盖原始传输或心跳证据。

## 测试策略

所有生产代码修改遵循测试先行。

1. HTTP 传输测试：TLS 握手失败且未触发 `WroteRequest`，必须分类为 definitely-not-sent。
2. HTTP 传输测试：服务端读取请求后不响应，必须分类为 request-written/unknown。
3. dispatcher 测试：pre-send 失败进入 `rejected`，不阻塞严格链。
4. dispatcher 测试：post-write 超时进入 `sent_unknown`，阻塞严格链且不重试。
5. PostgreSQL 集成测试：应用时钟分别快/慢数据库 2 秒，StartAttempt 和 heartbeat 仍能正确续租。
6. PostgreSQL 集成测试：错误 fencing token 无法续租。
7. 回归测试：正常接单、明确业务拒绝、期号不一致和 accepted finalizer 行为保持不变。
8. 全量执行 `go test ./...`、`go build ./cmd/server`、相关包 `go vet` 和 `git diff --check`。

## 发布与恢复步骤

1. 完成测试和代码审查后提交并推送两个远端。
2. `192.168.20.2` 拉取代码并重启后端。
3. 确认运行提交哈希、单一 8080 监听进程、NATS 和 64 个 dispatcher lease 正常。
4. 根据用户已经核对的第三方历史，将 outbox `606` 人工处理为 `rejected`，理由明确记录为第三方无对应订单。
5. 仅 rearm `inst-1-1787058821216`，使用现有 `0.20 USDT` 首注观察一次。
6. 必须同时看到第三方订单号、实际期号和本地 accepted/finalized 记录后，才扩大验证范围。
7. 若请求体写出后仍无法确认，保持阻塞并人工对账，不自动重复下注。

## 验收标准

- TLS 握手失败不会再产生 `external_acceptance_unknown`。
- 请求体可能已写出时绝不会自动重试或自动进入下一期。
- 后端与数据库存在至少 2 秒测试时钟偏差时，活跃请求 lease 不会误失效。
- 原始网络错误和心跳错误可通过管理员诊断接口读取。
- 正常第三方接单能在当前安全窗口内完成 accepted/finalized，并生成唯一投注记录。
- 单个故障方案不会影响其他方案、彩种或 dispatcher 分片。
