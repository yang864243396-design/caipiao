-- +goose Up
-- +goose StatementBegin
INSERT INTO chase_orders (
    chase_no, member_id, lottery_code, lottery_name, lottery_category,
    total_issues, done_issues, amount, status, started_at, finished_at, created_at
)
SELECT
    'C20260519000421', m.id, 'tencent_ffc', '时时彩 A', 'ssc',
    10, 3, 300.00, 'running', TIMESTAMPTZ '2026-05-19 02:12:33+00', NULL,
    TIMESTAMPTZ '2026-05-19 02:12:33+00'
FROM members m WHERE m.member_no = 'M00001'
ON CONFLICT (chase_no) DO NOTHING;

INSERT INTO chase_orders (
    chase_no, member_id, lottery_code, lottery_name, lottery_category,
    total_issues, done_issues, amount, status, started_at, finished_at, created_at
)
SELECT
    'C20260518000987', m.id, 'pk10_fast', 'PK10 快开', 'pk10',
    5, 5, 80.00, 'completed', TIMESTAMPTZ '2026-05-18 13:05:01+00',
    TIMESTAMPTZ '2026-05-18 15:20:00+00', TIMESTAMPTZ '2026-05-18 13:05:01+00'
FROM members m WHERE m.member_no = 'M00001'
ON CONFLICT (chase_no) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM chase_orders WHERE chase_no IN ('C20260519000421', 'C20260518000987');
-- +goose StatementEnd
