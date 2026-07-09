-- +goose Up
-- +goose StatementBegin
INSERT INTO copy_hall_rank_slots (lottery_code, board_kind, rank, scheme_id, scheme_name, play_method)
SELECT
    lot.code,
    b.board_kind,
    s.rank,
    CASE
        WHEN s.rank = 1 AND b.board_kind = 'master' THEN 'copy_demo_3001'
        WHEN b.board_kind = 'master' THEN 'copy_demo_' || (3000 + s.rank)::text
        WHEN s.rank = 1 AND b.board_kind = 'contrary' THEN 'copy_contrary_3001'
        ELSE 'copy_contrary_' || (3000 + s.rank)::text
    END,
    s.scheme_name,
    s.play_method
FROM (
    VALUES
        ('tencent_ffc'),
        ('cq_ssc'),
        ('xj_ssc'),
        ('tj_ssc'),
        ('fc_3d'),
        ('pl3')
) AS lot(code)
CROSS JOIN (VALUES ('master'), ('contrary')) AS b(board_kind)
CROSS JOIN (
    VALUES
        (1, '太乙后二', '定位胆万位'),
        (2, '紫燕万位', '定位胆后二'),
        (3, '莺凤十位', '定位胆十位'),
        (4, '宛天个位', '定位胆个位'),
        (5, '路线6000+', '组选六'),
        (6, '打狗前二', '定位胆前三'),
        (7, '邯肖任四', '任选四'),
        (8, '关冲70+', '定位胆后一'),
        (9, '猎豹后二', '定位胆千位'),
        (10, '青衫万位', '定位胆任二')
) AS s(rank, scheme_name, play_method)
WHERE b.board_kind = 'master'

UNION ALL

SELECT
    lot.code,
    'contrary',
    s.rank,
    CASE
        WHEN s.rank = 1 THEN 'copy_contrary_3001'
        ELSE 'copy_contrary_' || (3000 + s.rank)::text
    END,
    s.scheme_name,
    s.play_method
FROM (
    VALUES
        ('tencent_ffc'),
        ('cq_ssc'),
        ('xj_ssc'),
        ('tj_ssc'),
        ('fc_3d'),
        ('pl3')
) AS lot(code)
CROSS JOIN (
    VALUES
        (1, '逆锋万位', '定位胆万位'),
        (2, '反打后二', '定位胆后二'),
        (3, '折戟十位', '定位胆十位'),
        (4, '回风个位', '定位胆个位'),
        (5, '暗线3000-', '组选六'),
        (6, '退守前三', '定位胆前三'),
        (7, '虚晃任四', '任选四'),
        (8, '蛰伏50-', '定位胆后一'),
        (9, '裂空后一', '定位胆千位'),
        (10, '寒江千位', '定位胆任二')
) AS s(rank, scheme_name, play_method);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM copy_hall_rank_slots;
-- +goose StatementEnd
