-- +goose Up
-- +goose StatementBegin

-- Audit logs become append-only and tamper-evident:
--   * UPDATE, DELETE and TRUNCATE on audit_logs are rejected by triggers;
--   * each row stores the SHA-256 hash of its content chained to the previous
--     row's hash, so any later modification breaks the chain and is reported by
--     verify_audit_log_chain().
-- A superuser can still disable triggers and rewrite the whole chain; detecting
-- that requires anchoring the latest hash outside the database.

-- seq is the chain position. It is drawn inside the insert trigger while the
-- chain lock is held, so chain order always equals lock order.
CREATE SEQUENCE audit_logs_chain_seq;

ALTER TABLE audit_logs
    ADD COLUMN seq BIGINT,
    ADD COLUMN prev_hash TEXT,
    ADD COLUMN row_hash TEXT;

-- Actor references must not be rewritten to NULL by a cascade, because that
-- would be an UPDATE on an append-only table.
ALTER TABLE audit_logs
    DROP CONSTRAINT audit_logs_actor_user_id_fkey,
    DROP CONSTRAINT audit_logs_actor_org_id_fkey,
    ADD CONSTRAINT audit_logs_actor_user_id_fkey FOREIGN KEY (actor_user_id) REFERENCES users(id) ON DELETE RESTRICT,
    ADD CONSTRAINT audit_logs_actor_org_id_fkey FOREIGN KEY (actor_org_id) REFERENCES organizations(id) ON DELETE RESTRICT;

CREATE FUNCTION audit_log_row_hash(r audit_logs, prev TEXT) RETURNS TEXT AS $$
    SELECT encode(sha256(convert_to(concat_ws('|',
        r.id::text,
        COALESCE(r.request_id, ''),
        COALESCE(r.actor_user_id::text, ''),
        COALESCE(r.actor_org_id::text, ''),
        r.action,
        r.entity_type,
        COALESCE(r.entity_id::text, ''),
        r.before_data::text,
        r.after_data::text,
        COALESCE(r.ip_address, ''),
        COALESCE(r.user_agent, ''),
        (EXTRACT(EPOCH FROM r.created_at) * 1000000)::BIGINT::text,
        COALESCE(prev, '')
    ), 'UTF8')), 'hex');
$$ LANGUAGE sql IMMUTABLE;

-- Chain existing rows in insertion order.
DO $$
DECLARE
    r audit_logs;
    prev TEXT := NULL;
BEGIN
    FOR r IN SELECT * FROM audit_logs ORDER BY created_at, id LOOP
        UPDATE audit_logs
        SET seq = nextval('audit_logs_chain_seq'), prev_hash = prev, row_hash = audit_log_row_hash(r, prev)
        WHERE id = r.id;
        prev := audit_log_row_hash(r, prev);
    END LOOP;
END $$;

ALTER TABLE audit_logs
    ALTER COLUMN seq SET NOT NULL,
    ALTER COLUMN row_hash SET NOT NULL;
CREATE UNIQUE INDEX idx_audit_logs_seq ON audit_logs(seq);

CREATE FUNCTION audit_logs_chain_insert() RETURNS TRIGGER AS $$
DECLARE
    prev TEXT;
BEGIN
    -- Serialize inserts so concurrent transactions cannot fork the chain.
    PERFORM pg_advisory_xact_lock(hashtext('audit_logs_chain'));
    NEW.seq := nextval('audit_logs_chain_seq');
    SELECT row_hash INTO prev FROM audit_logs ORDER BY seq DESC LIMIT 1;
    NEW.prev_hash := prev;
    NEW.row_hash := audit_log_row_hash(NEW, prev);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE FUNCTION audit_logs_reject_change() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'audit_logs is append-only: % is not allowed', TG_OP
        USING ERRCODE = 'insufficient_privilege';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_logs_chain BEFORE INSERT ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION audit_logs_chain_insert();
CREATE TRIGGER audit_logs_no_update BEFORE UPDATE OR DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION audit_logs_reject_change();
CREATE TRIGGER audit_logs_no_truncate BEFORE TRUNCATE ON audit_logs
    FOR EACH STATEMENT EXECUTE FUNCTION audit_logs_reject_change();

-- Returns the first row whose stored hash or link no longer matches its
-- content; an empty result means the chain is intact.
CREATE FUNCTION verify_audit_log_chain()
RETURNS TABLE (broken_seq BIGINT, broken_id UUID, reason TEXT) AS $$
DECLARE
    r audit_logs;
    prev TEXT := NULL;
BEGIN
    FOR r IN SELECT * FROM audit_logs ORDER BY seq LOOP
        IF r.prev_hash IS DISTINCT FROM prev THEN
            RETURN QUERY SELECT r.seq, r.id, 'prev_hash does not match the preceding row'::TEXT;
            RETURN;
        END IF;
        IF r.row_hash <> audit_log_row_hash(r, prev) THEN
            RETURN QUERY SELECT r.seq, r.id, 'row content does not match row_hash'::TEXT;
            RETURN;
        END IF;
        prev := r.row_hash;
    END LOOP;
END;
$$ LANGUAGE plpgsql STABLE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION verify_audit_log_chain();
DROP TRIGGER audit_logs_no_truncate ON audit_logs;
DROP TRIGGER audit_logs_no_update ON audit_logs;
DROP TRIGGER audit_logs_chain ON audit_logs;
DROP FUNCTION audit_logs_reject_change();
DROP FUNCTION audit_logs_chain_insert();
DROP INDEX idx_audit_logs_seq;
ALTER TABLE audit_logs
    DROP CONSTRAINT audit_logs_actor_user_id_fkey,
    DROP CONSTRAINT audit_logs_actor_org_id_fkey,
    ADD CONSTRAINT audit_logs_actor_user_id_fkey FOREIGN KEY (actor_user_id) REFERENCES users(id) ON DELETE SET NULL,
    ADD CONSTRAINT audit_logs_actor_org_id_fkey FOREIGN KEY (actor_org_id) REFERENCES organizations(id) ON DELETE SET NULL;
DROP FUNCTION audit_log_row_hash(audit_logs, TEXT);
ALTER TABLE audit_logs
    DROP COLUMN row_hash,
    DROP COLUMN prev_hash,
    DROP COLUMN seq;
DROP SEQUENCE audit_logs_chain_seq;
-- +goose StatementEnd
