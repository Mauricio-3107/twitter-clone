package models

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

type Bookmark struct {
	ID            int
	TweetID       int
	UsernameLower string
}

type BookmarkService struct {
	DB *sql.DB
}

func (service *BookmarkService) Create(tweetID int, usernameLower string) (*Bookmark, error) {
	bookmark := Bookmark{
		TweetID:       tweetID,
		UsernameLower: usernameLower,
	}
	row := service.DB.QueryRow(`
	  INSERT INTO bookmarks (tweet_id, user_username)
	  VALUES ($1, $2) RETURNING id;`, tweetID, usernameLower)
	err := row.Scan(&bookmark.ID)
	if err != nil {
		return nil, fmt.Errorf("create bookmark: %w", err)
	}
	return &bookmark, nil
}

func (service *BookmarkService) GetExistingBookmark(tweetID int, usernameLower string) (bool, error) {
	var bookmarked bool
	row := service.DB.QueryRow(`
		SELECT EXISTS (SELECT 1 FROM bookmarks WHERE tweet_id = $1 AND user_username = $2);`, tweetID, usernameLower)
	err := row.Scan(&bookmarked)
	if err != nil {
		return bookmarked, fmt.Errorf("get existing bookmark: %w", err)
	}
	return bookmarked, nil
}

func (service *BookmarkService) Delete(tweetID int, usernameLower string) error {
	_, err := service.DB.Exec(`
	  DELETE FROM bookmarks
	  WHERE tweet_id = $1 AND user_username = $2;`, tweetID, usernameLower)
	if err != nil {
		return fmt.Errorf("delete bookmark: %w", err)
	}
	return nil
}

func (service *BookmarkService) ByUserNameLower(usernameLower string) ([]Tweet, []int, error) {

	var tweetIDs []int
	rows, err := service.DB.Query(`
	SELECT
		tweets.id,
		tweets.text,
		tweets.created_at,
		tweets.quoted_tweet_id,
		users.name,
		users.profile_image_url,
		users.username_original,
		ARRAY_AGG(tweet_images.image_url) AS tweet_images
	FROM
		tweets
	JOIN
		users ON tweets.user_username = users.username_lower
	LEFT JOIN
		tweet_images ON tweets.id = tweet_images.tweet_id
	JOIN
		bookmarks ON tweets.id = bookmarks.tweet_id
	WHERE
		bookmarks.user_username = $1
		AND tweets.parent_tweet_id = 0
	GROUP BY
		tweets.id,
		users.name,
		users.profile_image_url,
		users.username_original
	ORDER BY
		tweets.created_at DESC;`, usernameLower)
	if err != nil {
		return nil, tweetIDs, fmt.Errorf("query tweets by user ID: %w", err)
	}
	defer rows.Close()

	var tweets []Tweet
	for rows.Next() {
		var tweet Tweet
		var imagesURLs []sql.NullString
		err = rows.Scan(&tweet.ID, &tweet.Text, &tweet.CreatedAt, &tweet.QuotedTweetID, &tweet.Name, &tweet.ProfileImage, &tweet.UsernameOriginal, pq.Array(&imagesURLs))
		if err != nil {
			return nil, tweetIDs, fmt.Errorf("query bookmark tweets by user username: %w", err)
		}
		for _, imageURL := range imagesURLs {
			if imageURL.Valid {
				tweet.ImagesURL = append(tweet.ImagesURL, imageURL.String)
			}
		}

		tweets = append(tweets, tweet)
		tweetIDs = append(tweetIDs, tweet.ID)
	}
	if rows.Err() != nil {
		return nil, tweetIDs, fmt.Errorf("query tweets by user ID: %w", err)
	}
	return tweets, tweetIDs, nil
}
