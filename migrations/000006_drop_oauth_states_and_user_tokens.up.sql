-- Drop oauth_states and user_tokens tables (we store these in Redis now)
DROP TABLE IF EXISTS oauth_states;
DROP TABLE IF EXISTS user_tokens;
