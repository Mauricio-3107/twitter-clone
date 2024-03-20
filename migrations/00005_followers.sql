-- +goose Up
-- +goose StatementBegin
CREATE TABLE followers (
    id SERIAL PRIMARY KEY,
    user_username VARCHAR(255) REFERENCES users(username_lower) ON DELETE CASCADE,
    follower_username VARCHAR(255) REFERENCES users(username_lower) ON DELETE CASCADE,
    UNIQUE(user_username, follower_username)
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE followers;

-- +goose StatementEnd