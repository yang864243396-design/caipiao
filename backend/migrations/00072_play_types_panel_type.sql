-- +goose Up
-- P2：ssc_std 玩法类型 panel_type（驱动前端投注面板）
UPDATE play_types SET panel_type = 'dingwei' WHERE template_code = 'ssc_std' AND type_id = 'dingwei';
UPDATE play_types SET panel_type = 'segment' WHERE template_code = 'ssc_std' AND type_id IN (
    'qian3', 'zhong3', 'hou3', 'qian2', 'hou2', 'sixing', 'wuxing',
    'qianzhonghou3', 'qianhou3', 'combo24'
);
UPDATE play_types SET panel_type = 'longhu' WHERE template_code = 'ssc_std' AND type_id = 'longhu';
UPDATE play_types SET panel_type = 'renxuan' WHERE template_code = 'ssc_std' AND type_id = 'renxuan';
UPDATE play_types SET panel_type = 'textarea' WHERE template_code = 'ssc_std' AND type_id IN ('budingwei', 'dxds');

-- +goose Down
UPDATE play_types SET panel_type = NULL WHERE template_code = 'ssc_std';
