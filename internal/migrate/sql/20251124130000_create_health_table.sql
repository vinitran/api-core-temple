-- +goose Up
CREATE TABLE IF NOT EXISTS health_check (
    id          BIGSERIAL PRIMARY KEY,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    note        TEXT NOT NULL
);

INSERT INTO health_check (note) VALUES ('migrations initialized');

-- +goose Down
DROP TABLE IF EXISTS health_check;

