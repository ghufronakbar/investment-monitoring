-- +goose Up
-- +goose StatementBegin

-- Wake the outbox worker as soon as an event commits. PostgreSQL delivers
-- NOTIFY only after the inserting transaction commits, so the worker never sees
-- an event whose business change could still roll back. Polling remains as a
-- fallback for missed notifications and for poolers that do not support LISTEN.
CREATE FUNCTION outbox_events_notify() RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify('outbox_events', NEW.id::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER outbox_events_notify AFTER INSERT ON outbox_events
    FOR EACH ROW EXECUTE FUNCTION outbox_events_notify();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER outbox_events_notify ON outbox_events;
DROP FUNCTION outbox_events_notify();
-- +goose StatementEnd
