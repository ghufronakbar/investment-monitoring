-- +goose Up
-- +goose StatementBegin

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id TEXT,
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_org_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id UUID,
    before_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    after_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_actor_user ON audit_logs(actor_user_id, created_at DESC);
CREATE INDEX idx_audit_logs_actor_org ON audit_logs(actor_org_id, created_at DESC);
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id, created_at DESC);
CREATE INDEX idx_audit_logs_action ON audit_logs(action, created_at DESC);
CREATE INDEX idx_audit_logs_request_id ON audit_logs(request_id) WHERE request_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS audit_logs;

-- +goose StatementEnd
