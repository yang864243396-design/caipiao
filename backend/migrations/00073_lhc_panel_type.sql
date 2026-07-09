-- +goose Up
-- P3：lhc_std 玩法类型 panel_type（驱动六合彩投注面板）
UPDATE play_types SET panel_type = 'lhc_num' WHERE template_code = 'lhc_std' AND type_id IN (
    'tema', 'erquanzhong', 'erzhongte', 'techuan', 'sanzhonger', 'sanquanzhong',
    'buzhong_xuanyi', 'qima', 'renzhong'
);
UPDATE play_types SET panel_type = 'lhc_zodiac' WHERE template_code = 'lhc_std' AND type_id = 'shengxiao';
UPDATE play_types SET panel_type = 'lhc_tail' WHERE template_code = 'lhc_std' AND type_id = 'weishu';
UPDATE play_types SET panel_type = 'lhc_attr' WHERE template_code = 'lhc_std' AND type_id IN ('wuxingjiaye', 'bose', 'tematouwei');
UPDATE play_types SET panel_type = 'textarea' WHERE template_code = 'lhc_std' AND type_id = 'guoguan';

-- +goose Down
UPDATE play_types SET panel_type = NULL WHERE template_code = 'lhc_std';
