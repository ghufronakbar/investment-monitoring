-- +goose Up
-- +goose StatementBegin
ALTER TABLE payment_requests 
ADD COLUMN payment_proof_image_urls TEXT[] NOT NULL DEFAULT '{}';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE payment_requests 
DROP COLUMN payment_proof_image_urls;
-- +goose StatementEnd
