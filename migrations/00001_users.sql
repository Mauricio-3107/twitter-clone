-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    username_original VARCHAR(255) NOT NULL,
    username_lower VARCHAR(255) UNIQUE NOT NULL,
    date_of_birth DATE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    profile_image_url TEXT NOT NULL
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE users;

-- +goose StatementEnd