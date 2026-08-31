-- +goose Up
-- +goose StatementBegin

-- A service account is deactivated, never deleted, for the same reason a person
-- is: the journal names it, and evidence has to survive the account that
-- produced it (ADR 0018). 00006 gave `users` this column and left it off
-- `service_accounts`, which made retiring a program impossible without losing
-- what it pushed.
ALTER TABLE service_accounts ADD COLUMN deactivated_at timestamptz;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE service_accounts DROP COLUMN deactivated_at;

-- +goose StatementEnd
