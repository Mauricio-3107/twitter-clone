-- +goose Up
-- +goose StatementBegin
CREATE TABLE tweet_images (
    id SERIAL PRIMARY KEY,
    tweet_id INTEGER REFERENCES tweets(id) ON DELETE CASCADE,
    image_url TEXT NOT NULL
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE tweet_images;

-- +goose StatementEnd