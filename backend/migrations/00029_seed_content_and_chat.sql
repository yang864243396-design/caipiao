-- +goose Up
-- +goose StatementBegin
INSERT INTO cms_announcements (id, title, status, published_at, body_html) VALUES
('n1', '入款重要公告', 'published', TIMESTAMPTZ '2026-05-18 10:00:00+00',
 '<p>尊敬会员：入款前请确认渠道与限额，到账时间以第三方支付回调为准。</p><p>如有疑问请联系在线客服。</p>'),
('usdt-ton', '新增【USDT-TON】渠道公告', 'published', TIMESTAMPTZ '2026-05-16 08:00:00+00',
 '<p>平台已新增 <strong>USDT-TON</strong> 充值渠道，请确认网络类型后再转账。</p><p>错误网络将导致资金无法找回。</p>'),
('version-2-4', '平台投注规则郑重声明公告', 'published', TIMESTAMPTZ '2026-05-12 09:00:00+00',
 '<p>请遵守平台投注规则，理性参与。详细条款见帮助中心。</p>'),
('n2', '中信银行长期维护公告', 'published', TIMESTAMPTZ '2026-05-08 12:00:00+00',
 '<p>中信银行通道维护期间可能延迟到账，请优先使用其他渠道。</p>'),
('n3', 'USDT 出款公告', 'published', TIMESTAMPTZ '2026-05-01 10:00:00+00',
 '<p>USDT 出款需完成实名与风控校验，预计 1–30 分钟到账。</p>'),
('n4', '移动端体验优化说明', 'published', TIMESTAMPTZ '2026-04-22 08:00:00+00',
 '<p>会员中心与跟单大厅已完成移动端布局优化，建议更新至最新版本。</p>')
ON CONFLICT (id) DO NOTHING;

INSERT INTO cms_faq_articles (id, title, sort, body_html) VALUES
('legend-faq-user', '【传奇挂机】 常见问题(用户必看)', 1,
 '<p>挂机方案建议在稳定的网络环境下运行，并确保客户端版本为当前渠道推荐版本。</p><p>若出现方案未按预期执行，请先检查当期彩种休市时间及账户资金是否充足，再联系在线客服提供注单号协助排查。</p>'),
('legend-steps', '【传奇挂机】 简单操作步骤手册', 2,
 '<p>① 在「方案库」选择或新建挂机方案；② 绑定彩种与期号规则；③ 设置单注金额与止损上限；④ 一键启动并可在运行面板查看状态。</p>'),
('legend-notes', '【传奇挂机】 操作注意事项', 3,
 '<p>请勿在多设备同时启动同一方案的自动投注，可能造成重复下单。</p>'),
('ssc-multiplier-guide', '时时彩倍投的方案全面讲解如何倍投', 4,
 '<p>倍投的本质是提高后续期次的单位注额以覆盖前期成本，需严格设定最大期数与单方案资金上限。</p>'),
('software-prev-period-fix', '软件老是购买到上期号码的解决方案', 5,
 '<p>多为本地时间与服务端期号不同步导致，请在设置中开启「以服务器期号为准」并刷新页面。</p>'),
('ssc-behaviors-avoid', '购买时彩必须要拒绝的N种行为', 6,
 '<p>拒绝无计划追高、在情绪化状态下加注、混用多套互斥规则不清晰方案等行为。</p>'),
('four-years-high-freq-summary', '四年玩高频彩下来的经验亲身总结...', 7,
 '<p>高频玩法节奏快，更需要记录每笔策略与复盘；长期看，纪律比「灵感」更重要。</p>')
ON CONFLICT (id) DO NOTHING;

INSERT INTO cms_help_articles (id, title, sort, body_html) VALUES
('terms', '条款与规则', 1,
 '<p>使用本平台前，请仔细阅读服务条款、隐私政策及投注规则。继续访问即视为您已理解并同意相关约束。</p><p class="muted">完整法律文本以平台公示为准；如有更新，将以公告形式通知。</p>'),
('responsible', '博彩责任', 2,
 '<p>请理性参与，仅在个人可承受范围内娱乐；切勿借贷投注或影响正常生活与工作。</p><p>若您感到难以自控，请暂停使用并寻求专业帮助。</p>'),
('faq-intro', '常见问题', 3,
 '<p>汇总高频操作说明、挂机与倍投说明、客户端异常排查等，按主题分类浏览。</p>')
ON CONFLICT (id) DO NOTHING;

INSERT INTO member_system_messages (member_id, body, created_at)
SELECT m.id, v.body, v.created_at
FROM members m
CROSS JOIN (VALUES
    ('您使用JJ支付的入款申请(9999元)已被取消，谢谢。', TIMESTAMPTZ '2026-05-18 11:54:25+00'),
    ('您使用JJ支付的入款申请(499元)已被取消，谢谢。', TIMESTAMPTZ '2026-05-18 10:12:08+00'),
    ('【演示】您的周返利 128.60 元已发放至中心钱包，请留意可用余额变动。', TIMESTAMPTZ '2026-05-17 09:00:02+00'),
    ('赛事公告：今夜 22:00 起部分足球盘口将提前封盘维护，预计 30 分钟。', TIMESTAMPTZ '2026-05-16 18:45:00+00')
) AS v(body, created_at)
WHERE m.account = 'vs8888';

INSERT INTO member_chat_messages (member_id, peer_key, direction, body, created_at)
SELECT m.id, v.peer_key, v.direction, v.body, v.created_at
FROM members m
CROSS JOIN (VALUES
    ('service', 'in', '您好，请问有什么可以帮您？', TIMESTAMPTZ '2026-05-18 09:00:00+00'),
    ('superior', 'in', '本周团队业绩已更新，请查看团队总览。', TIMESTAMPTZ '2026-05-17 14:30:00+00'),
    ('notice-deposit', 'in', '入款审核已通过', TIMESTAMPTZ '2026-05-18 08:00:00+00'),
    ('notice-bonus', 'in', '周返利已到账', TIMESTAMPTZ '2026-05-17 09:05:00+00')
) AS v(peer_key, direction, body, created_at)
WHERE m.account = 'vs8888';

INSERT INTO member_announcement_reads (member_id, announcement_id, read_at)
SELECT m.id, 'version-2-4', TIMESTAMPTZ '2026-05-13 10:00:00+00'
FROM members m WHERE m.account = 'vs8888'
ON CONFLICT DO NOTHING;

INSERT INTO member_announcement_reads (member_id, announcement_id, read_at)
SELECT m.id, 'n3', TIMESTAMPTZ '2026-05-02 10:00:00+00'
FROM members m WHERE m.account = 'vs8888'
ON CONFLICT DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM member_announcement_reads
WHERE announcement_id IN ('version-2-4', 'n3')
  AND member_id IN (SELECT id FROM members WHERE account = 'vs8888');

DELETE FROM member_chat_messages
WHERE member_id IN (SELECT id FROM members WHERE account = 'vs8888');

DELETE FROM member_system_messages
WHERE member_id IN (SELECT id FROM members WHERE account = 'vs8888');

DELETE FROM cms_help_articles WHERE id IN ('terms', 'responsible', 'faq-intro');
DELETE FROM cms_faq_articles WHERE id IN (
    'legend-faq-user', 'legend-steps', 'legend-notes', 'ssc-multiplier-guide',
    'software-prev-period-fix', 'ssc-behaviors-avoid', 'four-years-high-freq-summary'
);
DELETE FROM cms_announcements WHERE id IN ('n1', 'usdt-ton', 'version-2-4', 'n2', 'n3', 'n4');
-- +goose StatementEnd
