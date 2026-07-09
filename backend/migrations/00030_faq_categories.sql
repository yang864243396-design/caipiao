-- +goose Up
-- +goose StatementBegin
CREATE TABLE cms_faq_categories (
    id         VARCHAR(64)  PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    sort       INT          NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE cms_faq_categories IS 'FAQ 分类（Admin 维护）';

ALTER TABLE cms_faq_articles
    ADD COLUMN category_id VARCHAR(64) REFERENCES cms_faq_categories(id) ON DELETE SET NULL;

COMMENT ON COLUMN cms_faq_articles.category_id IS '所属 FAQ 分类';

INSERT INTO cms_faq_categories (id, name, sort) VALUES
('FC1', '账户与资金', 1),
('FC2', '玩法与投注', 2),
('FC3', '代理与团队', 3)
ON CONFLICT (id) DO NOTHING;

UPDATE cms_faq_articles SET category_id = 'FC2' WHERE id IN (
    'legend-faq-user', 'legend-steps', 'legend-notes',
    'ssc-multiplier-guide', 'software-prev-period-fix', 'ssc-behaviors-avoid'
);
UPDATE cms_faq_articles SET category_id = 'FC3' WHERE id = 'four-years-high-freq-summary';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE cms_faq_articles DROP COLUMN IF EXISTS category_id;
DROP TABLE IF EXISTS cms_faq_categories;
-- +goose StatementEnd
