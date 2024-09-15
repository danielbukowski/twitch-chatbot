-- +goose Up
-- +goose StatementBegin
CREATE TABLE access_credentials (
    access_credential_id INTEGER,
    channel_name TEXT UNIQUE NOT NULL,
    details TEXT NOT NULL,
    PRIMARY KEY (access_credential_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE access_credentials;
-- +goose StatementEnd
