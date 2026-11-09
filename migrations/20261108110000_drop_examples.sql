-- +goose Up
-- +goose StatementBegin
-- Remove the scaffold table left over from the project template.
DROP TABLE examples;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE examples (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    price BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE TRIGGER update_examples_modtime BEFORE UPDATE ON examples FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
-- +goose StatementEnd
