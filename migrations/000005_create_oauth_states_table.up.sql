CREATE TABLE oauth_states(
    states          UUID PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform        TEXT,
    intent          TEXT,
    created_at      TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ
)