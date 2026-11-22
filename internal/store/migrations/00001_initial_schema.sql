-- +goose Up
CREATE TABLE users (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- Stored lowercased; the application normalizes before every read and write.
    email               TEXT        NOT NULL,
    name                TEXT        NOT NULL,
    password_hash       TEXT        NOT NULL,
    role                TEXT        NOT NULL DEFAULT 'member',
    is_active           BOOLEAN     NOT NULL DEFAULT TRUE,
    -- Sessions record this value at login and are rejected once it changes.
    password_changed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT users_email_key UNIQUE (email),
    CONSTRAINT users_email_lowercase CHECK (email = lower(email)),
    CONSTRAINT users_role_valid CHECK (role IN ('admin', 'member'))
);

-- Session storage for alexedwards/scs (pgxstore).
CREATE TABLE sessions (
    token  TEXT        PRIMARY KEY,
    data   BYTEA       NOT NULL,
    expiry TIMESTAMPTZ NOT NULL
);

CREATE INDEX sessions_expiry_idx ON sessions (expiry);

-- +goose Down
DROP TABLE sessions;
DROP TABLE users;
