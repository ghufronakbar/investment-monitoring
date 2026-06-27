-- +goose Up
-- +goose StatementBegin

CREATE TABLE sla_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,
    state TEXT NOT NULL,
    owner_org_id UUID REFERENCES organizations(id) ON DELETE RESTRICT,
    entered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    due_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'ON_TRACK',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT sla_records_status_check CHECK (status IN ('ON_TRACK', 'SLA_WARNING', 'SLA_VIOLATED', 'RESOLVED')),
    CONSTRAINT sla_records_entity_unique UNIQUE (entity_type, entity_id)
);

CREATE INDEX idx_sla_records_entity ON sla_records(entity_type, entity_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_sla_records_due_at ON sla_records(due_at) WHERE deleted_at IS NULL AND resolved_at IS NULL;
CREATE INDEX idx_sla_records_owner ON sla_records(owner_org_id) WHERE deleted_at IS NULL;

CREATE TRIGGER update_sla_records_modtime BEFORE UPDATE ON sla_records FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS sla_records;

-- +goose StatementEnd
