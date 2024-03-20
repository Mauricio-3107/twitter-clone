-- +goose Up
-- +goose StatementBegin
CREATE TABLE following (
    id SERIAL PRIMARY KEY,
    user_username VARCHAR(255) REFERENCES users(username_lower) ON DELETE CASCADE,
    following_username VARCHAR(255) REFERENCES users(username_lower) ON DELETE CASCADE,
    UNIQUE(user_username, following_username) -- Ensure no duplicate followings for a user
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE following;

-- +goose StatementEnd