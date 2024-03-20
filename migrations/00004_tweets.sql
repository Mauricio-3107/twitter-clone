-- +goose Up
-- +goose StatementBegin
CREATE TABLE tweets (
    id SERIAL PRIMARY KEY,
    user_username VARCHAR(255) REFERENCES users(username_lower) ON DELETE CASCADE,
    text TEXT CHECK (LENGTH(text) <= 280),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    parent_tweet_id INT DEFAULT 0,
    quoted_tweet_id INT DEFAULT 0
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE tweets;

-- +goose StatementEnd