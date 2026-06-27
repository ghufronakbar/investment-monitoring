-- +goose Up
-- +goose StatementBegin

CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    public_id TEXT NOT NULL UNIQUE,
    url TEXT NOT NULL,
    file_name TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size BIGINT NOT NULL,
    checksum TEXT NOT NULL,
    uploaded_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    uploaded_by_org_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    entity_type TEXT,
    entity_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT files_size_positive_check CHECK (size > 0),
    CONSTRAINT files_entity_pair_check CHECK (
        (entity_type IS NULL AND entity_id IS NULL)
        OR (entity_type IS NOT NULL AND entity_id IS NOT NULL)
    )
);

CREATE TRIGGER update_files_modtime BEFORE UPDATE ON files FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

CREATE INDEX idx_files_uploaded_by_user ON files(uploaded_by_user_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_files_uploaded_by_org ON files(uploaded_by_org_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_files_entity ON files(entity_type, entity_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_files_checksum ON files(checksum) WHERE deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS files;

-- +goose StatementEnd
