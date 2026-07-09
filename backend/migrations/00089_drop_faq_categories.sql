-- +goose Up
-- +goose StatementBegin
ALTER TABLE cms_faq_articles DROP COLUMN IF EXISTS category_id;
DROP TABLE IF EXISTS cms_faq_categories;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE cms_faq_categories (
    id         VARCHAR(64)  PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    sort       INT          NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

ALTER TABLE cms_faq_articles
    ADD COLUMN category_id VARCHAR(64) REFERENCES cms_faq_categories(id) ON DELETE SET NULL;
-- +goose StatementEnd
