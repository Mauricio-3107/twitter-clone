-- +goose Up
-- +goose StatementBegin
CREATE TABLE bookmarks (
    id SERIAL PRIMARY KEY,
    tweet_id INTEGER REFERENCES tweets(id) ON DELETE CASCADE,
    user_username VARCHAR(255) REFERENCES users(username_lower) ON DELETE CASCADE,
    UNIQUE(tweet_id, user_username)
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE bookmarks;

-- +goose StatementEnd