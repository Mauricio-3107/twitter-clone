package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

type TweetImage struct {
	ID       int
	TweetID  int
	ImageURL string
}

type TweetLike struct {
	ID            int
	TweetID       int
	UsernameLower string
}

type Retweet struct {
	ID            int
	TweetID       int
	UsernameLower string
}

type Tweet struct {
	ID               int
	Text             string
	CreatedAt        time.Time
	UsernameOriginal string
	UsernameLower    string
	Name             string
	ProfileImage     string
	Images           []TweetImage
	ImagesURL        []string
	Retweeted        bool
	ParentTweetID    int // Indicates the parent tweet ID for replies, 0 for original tweets
	QuotedTweetID    int // If not zero, is a quoted tweet
}

type TweetService struct {
	DB               *sql.DB
	ImageUserService ImageService
}

func (service *TweetService) Create(text, usernameOriginal, name, profileImg string, images bool, parentTweetID, quotedTweetID int) (*Tweet, error) {
	if len(text) < 1 && !images {
		// no text and no image and the form must not be recieved
		return nil, ErrEmptyTweet
	} else if len(text) > 280 {
		return nil, ErrLimitMaxText
	}

	// Normalize usernameOriginal and followingUsername (lowecase)
	userUsername := strings.ToLower(usernameOriginal)

	// Tweet or Reply
	var parentTweetIDValue int
	if parentTweetID == 0 {
		parentTweetIDValue = 0
	} else {
		parentTweetIDValue = parentTweetID
	}

	// Tweet or Quoted tweet
	var quotedTweetIDValue int
	if quotedTweetID == 0 {
		quotedTweetIDValue = 0
	} else {
		quotedTweetIDValue = quotedTweetID
	}

	tweet := Tweet{
		Text:             text,
		CreatedAt:        time.Now(),
		UsernameLower:    userUsername,
		UsernameOriginal: usernameOriginal,
		Name:             name,
		ProfileImage:     profileImg,
		ParentTweetID:    parentTweetIDValue,
		QuotedTweetID:    quotedTweetIDValue,
	}
	row := service.DB.QueryRow(`
	  INSERT INTO tweets (user_username, text, created_at, parent_tweet_id, quoted_tweet_id)
	  VALUES ($1, $2, $3, $4, $5) RETURNING id`, tweet.UsernameLower, tweet.Text, tweet.CreatedAt, tweet.ParentTweetID, tweet.QuotedTweetID)
	err := row.Scan(&tweet.ID)
	if err != nil {
		return nil, fmt.Errorf("create tweet: %w", err)
	}
	return &tweet, nil
}

func (service *TweetService) CreateTweetImages(tweet Tweet, tweetImages ...string) (*Tweet, error) {
	// Normalize the quantity of images
	if len(tweetImages) > 4 {
		return nil, ErrLimitMaxImages
	}

	// Save the url images
	for _, imgURL := range tweetImages {
		tweetImage := TweetImage{
			TweetID:  tweet.ID,
			ImageURL: imgURL,
		}
		// /images/tweets/{tweetID}/photo-{idx}
		// Images are already in the URL format
		row := service.DB.QueryRow(`
	  		INSERT INTO tweet_images (tweet_id, image_url)
	  		VALUES ($1, $2) RETURNING id`, tweetImage.TweetID, tweetImage.ImageURL)
		err := row.Scan(&tweetImage.ID)
		if err != nil {
			return nil, fmt.Errorf("create tweet image: %w", err)
		}
		tweet.Images = append(tweet.Images, tweetImage)
	}
	return &tweet, nil
}

func (service *TweetService) ByTweetID(tweetID int) (*Tweet, []*Tweet, error) {
	tweet := Tweet{
		ID: tweetID,
	}
	row := service.DB.QueryRow(`
	  SELECT tweets.user_username, tweets.text, tweets.created_at, tweets.parent_tweet_id, tweets.quoted_tweet_id, users.username_original, users.name, users.profile_image_url
	  FROM tweets
	  JOIN users ON tweets.user_username = users.username_lower
	  WHERE tweets.id= $1;`, tweet.ID)
	err := row.Scan(&tweet.UsernameLower, &tweet.Text, &tweet.CreatedAt, &tweet.ParentTweetID, &tweet.QuotedTweetID, &tweet.UsernameOriginal, &tweet.Name, &tweet.ProfileImage)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, fmt.Errorf("query tweet by id: %w", err)
	}
	// Fetch image URLs associated with the tweet
	rows, err := service.DB.Query(`
		SELECT image_url
		FROM tweet_images
		WHERE tweet_id = $1;`, tweetID)
	if err != nil {
		return nil, nil, fmt.Errorf("query tweet images by tweet ID: %w", err)
	}
	defer rows.Close()
	// Iterate over the rows and append image URLs to the tweet's image URLs slice
	for rows.Next() {
		var imageURL sql.NullString
		err := rows.Scan(&imageURL)
		if err != nil {
			return nil, nil, fmt.Errorf("scan tweet image URL: %w", err)
		}
		if imageURL.Valid {
			tweet.ImagesURL = append(tweet.ImagesURL, imageURL.String)
		}

	}
	if rows.Err() != nil {
		return nil, nil, fmt.Errorf("iterate over tweet image rows: %w", err)
	}

	var replies []*Tweet
	replies, err = service.ReplyTweets(tweetID)
	if err != nil {
		return nil, nil, fmt.Errorf("query by tweet ID: %w", err)
	}

	return &tweet, replies, nil
}

// func (service *TweetService) FeedWithUserTweets(usernameOriginal string) ([]Tweet, []int, error) {
// 	// Normalize usernameOriginal (lowercase)
// 	username := strings.ToLower(usernameOriginal)

// 	var tweetIDs []int

// 	rows, err := service.DB.Query(`
// 		SELECT
// 			tweets.id,
// 			tweets.text,
// 			tweets.created_at,
// 			users.username_original,
// 			users.name,
// 			users.profile_image_url,
// 			ARRAY_AGG(tweet_images.image_url)
// 		FROM
// 			tweets
// 			JOIN users ON tweets.user_username = users.username_lower
// 			LEFT JOIN tweet_images ON tweets.id = tweet_images.tweet_id
// 		WHERE
// 			(tweets.user_username = $1 OR tweets.user_username IN (
// 				SELECT
// 					following.following_username
// 				FROM
// 					following
// 				WHERE
// 					following.user_username = $1
// 			)) AND tweets.parent_tweet_id = 0
// 		GROUP BY
// 			tweets.id,
// 			users.username_original,
// 			users.name,
// 			users.profile_image_url
// 		ORDER BY
// 			tweets.created_at DESC;`, username)

// 	if err != nil {
// 		return nil, tweetIDs, fmt.Errorf("query feed by user's following and own tweets: %w", err)
// 	}
// 	defer rows.Close()

// 	var tweets []Tweet

// 	for rows.Next() {
// 		var tweet Tweet
// 		var imageURLs []sql.NullString
// 		err := rows.Scan(&tweet.ID, &tweet.Text, &tweet.CreatedAt, &tweet.UsernameOriginal, &tweet.Name, &tweet.ProfileImage, pq.Array(&imageURLs))
// 		if err != nil {
// 			return nil, tweetIDs, fmt.Errorf("scan tweet row micho: %w", err)
// 		}
// 		// Iterate over image URLs and add valid ones to tweet.ImagesURL
// 		for _, imageURL := range imageURLs {
// 			if imageURL.Valid {
// 				tweet.ImagesURL = append(tweet.ImagesURL, imageURL.String)
// 			}
// 		}

// 		tweets = append(tweets, tweet)
// 		tweetIDs = append(tweetIDs, tweet.ID)
// 	}
// 	if rows.Err() != nil {
// 		return nil, tweetIDs, fmt.Errorf("iterate over tweet rows: %w", rows.Err())
// 	}
// 	return tweets, tweetIDs, nil
// }

func (service *TweetService) FeedWithUserTweets(usernameOriginal string) ([]Tweet, []int, error) {
	// Normalize usernameOriginal (lowercase)
	username := strings.ToLower(usernameOriginal)

	var tweetIDs []int

	rows, err := service.DB.Query(`
	SELECT
		tweets.id,
		tweets.text,
		tweets.created_at,
		tweets.quoted_tweet_id,
		users.username_original,
		users.name,
		users.profile_image_url,
		ARRAY_AGG(tweet_images.image_url),
		false AS is_retweet
	FROM
		tweets
		JOIN users ON tweets.user_username = users.username_lower
		LEFT JOIN tweet_images ON tweets.id = tweet_images.tweet_id
	WHERE
		(
			tweets.user_username = $1
			OR tweets.user_username IN (
				SELECT
					following.following_username
				FROM
					following
				WHERE
					following.user_username = $1
			)
		)
		AND tweets.parent_tweet_id = 0
	GROUP BY
		tweets.id,
		users.username_original,
		users.name,
		users.profile_image_url
	UNION
	ALL
	SELECT
		tweets.id,
		tweets.text,
		tweets.created_at,
		tweets.quoted_tweet_id,
		users.username_original,
		users.name,
		users.profile_image_url,
		ARRAY_AGG(tweet_images.image_url),
		true AS is_retweet
	FROM
		retweets
		JOIN tweets ON retweets.tweet_id = tweets.id
		JOIN users ON tweets.user_username = users.username_lower
		LEFT JOIN tweet_images ON tweets.id = tweet_images.tweet_id
	WHERE
		retweets.user_username = $1
		AND tweets.parent_tweet_id = 0
	GROUP BY
		tweets.id,
		users.username_original,
		users.name,
		users.profile_image_url
	ORDER BY
		created_at DESC;`, username)

	if err != nil {
		return nil, tweetIDs, fmt.Errorf("query feed by user's following and own tweets: %w", err)
	}
	defer rows.Close()

	var tweets []Tweet

	for rows.Next() {
		var tweet Tweet
		var imageURLs []sql.NullString
		err := rows.Scan(&tweet.ID, &tweet.Text, &tweet.CreatedAt, &tweet.QuotedTweetID, &tweet.UsernameOriginal, &tweet.Name, &tweet.ProfileImage, pq.Array(&imageURLs), &tweet.Retweeted)
		if err != nil {
			return nil, tweetIDs, fmt.Errorf("scan tweet row: %w", err)
		}
		// Iterate over image URLs and add valid ones to tweet.ImagesURL
		for _, imageURL := range imageURLs {
			if imageURL.Valid {
				tweet.ImagesURL = append(tweet.ImagesURL, imageURL.String)
			}
		}

		tweets = append(tweets, tweet)
		tweetIDs = append(tweetIDs, tweet.ID)
	}
	if rows.Err() != nil {
		return nil, tweetIDs, fmt.Errorf("iterate over tweet rows: %w", rows.Err())
	}
	return tweets, tweetIDs, nil
}

func (service *TweetService) ReplyTweets(tweetID int) ([]*Tweet, error) {
	rows, err := service.DB.Query(`
		SELECT
			tweets.id,
			tweets.text,
			tweets.created_at,
			users.username_original,
			users.name,
			users.profile_image_url,
			ARRAY_AGG(tweet_images.image_url)
		FROM
			tweets
			JOIN users ON tweets.user_username = users.username_lower
			LEFT JOIN tweet_images ON tweets.id = tweet_images.tweet_id
		WHERE tweets.parent_tweet_id = $1
		GROUP BY
			tweets.id,
			users.username_original,
			users.name,
			users.profile_image_url
		ORDER BY
			tweets.created_at DESC;`, tweetID)

	if err != nil {
		return nil, fmt.Errorf("query replies tweetID: %w", err)
	}
	defer rows.Close()

	var tweets []*Tweet

	for rows.Next() {
		var tweet Tweet
		var imageURLs []sql.NullString
		err := rows.Scan(&tweet.ID, &tweet.Text, &tweet.CreatedAt, &tweet.UsernameOriginal, &tweet.Name, &tweet.ProfileImage, pq.Array(&imageURLs))
		if err != nil {
			return nil, fmt.Errorf("scan query replies tweetID: %w", err)
		}
		// Iterate over image URLs and add valid ones to tweet.ImagesURL
		for _, imageURL := range imageURLs {
			if imageURL.Valid {
				tweet.ImagesURL = append(tweet.ImagesURL, imageURL.String)
			}
		}

		tweets = append(tweets, &tweet)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate over replies rows: %w", rows.Err())
	}
	return tweets, nil
}

func (service *TweetService) ByUserNameOriginal(userUsernameOriginal string) ([]Tweet, []int, error) {
	// Normalize userUsername and followingUsername (lowecase)
	userUsername := strings.ToLower(userUsernameOriginal)

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
		ARRAY_AGG(tweet_images.image_url),
		false AS is_retweet
	FROM
		tweets
		JOIN users ON tweets.user_username = users.username_lower
		LEFT JOIN tweet_images ON tweets.id = tweet_images.tweet_id
	WHERE
		users.username_original = $1
		AND tweets.parent_tweet_id = 0
	GROUP BY
		tweets.id,
		users.name,
		users.profile_image_url,
		users.username_original
	UNION
	ALL
	SELECT
		tweets.id,
		tweets.text,
		tweets.created_at,
		tweets.quoted_tweet_id,
		users.name,
		users.profile_image_url,
		users.username_original,
		ARRAY_AGG(tweet_images.image_url),
		true AS is_retweet
	FROM
		retweets
		JOIN tweets ON retweets.tweet_id = tweets.id
		JOIN users ON tweets.user_username = users.username_lower
		LEFT JOIN tweet_images ON tweets.id = tweet_images.tweet_id
	WHERE
		retweets.user_username = $1
		AND tweets.parent_tweet_id = 0
	GROUP BY
		tweets.id,
		users.name,
		users.profile_image_url,
		users.username_original
	ORDER BY
		created_at DESC;`, userUsername)
	if err != nil {
		return nil, tweetIDs, fmt.Errorf("query tweets by user ID: %w", err)
	}
	defer rows.Close()

	var tweets []Tweet
	for rows.Next() {
		var tweet Tweet
		var imagesURLs []sql.NullString
		err = rows.Scan(&tweet.ID, &tweet.Text, &tweet.CreatedAt, &tweet.QuotedTweetID, &tweet.Name, &tweet.ProfileImage, &tweet.UsernameOriginal, pq.Array(&imagesURLs), &tweet.Retweeted)
		if err != nil {
			return nil, tweetIDs, fmt.Errorf("query tweets by user username original: %w", err)
		}
		for _, imageURL := range imagesURLs {
			if imageURL.Valid {
				tweet.ImagesURL = append(tweet.ImagesURL, imageURL.String)
			}
		}
		if userUsernameOriginal != tweet.UsernameOriginal {
			tweet.Retweeted = true
		}
		tweets = append(tweets, tweet)
		tweetIDs = append(tweetIDs, tweet.ID)
	}
	if rows.Err() != nil {
		return nil, tweetIDs, fmt.Errorf("query tweets by user ID: %w", err)
	}
	return tweets, tweetIDs, nil
}

/* func (service *TweetService) ByUserNameOriginal(userUsernameOriginal string) ([]Tweet, []int, error) {
	// Normalize userUsername and followingUsername (lowecase)
	userUsername := strings.ToLower(userUsernameOriginal)

	var tweetIDs []int
	rows, err := service.DB.Query(`
		SELECT
			tweets.id,
			tweets.text,
			tweets.created_at,
			users.name,
			users.profile_image_url,
			ARRAY_AGG(tweet_images.image_url)
		FROM
			tweets
			JOIN users ON tweets.user_username = users.username_lower
			LEFT JOIN tweet_images ON tweets.id = tweet_images.tweet_id
		WHERE
			users.username_original = $1 AND tweets.parent_tweet_id = 0
		GROUP BY
			tweets.id,
			users.name,
			users.profile_image_url
		ORDER BY
			tweets.created_at DESC;`, userUsername)
	if err != nil {
		return nil, tweetIDs, fmt.Errorf("query tweets by user ID: %w", err)
	}
	defer rows.Close()

	var tweets []Tweet
	for rows.Next() {
		tweet := Tweet{
			UsernameOriginal: userUsernameOriginal,
		}
		var imagesURLs []sql.NullString
		err = rows.Scan(&tweet.ID, &tweet.Text, &tweet.CreatedAt, &tweet.Name, &tweet.ProfileImage, pq.Array(&imagesURLs))
		if err != nil {
			return nil, tweetIDs, fmt.Errorf("query tweets by user username original: %w", err)
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
*/

func (service *TweetService) ProfileData(usernameOriginal string) (*User, error) {
	username := strings.ToLower(usernameOriginal)
	user := User{
		UsernameOriginal: usernameOriginal,
	}
	row := service.DB.QueryRow(`
	  SELECT users.name, users.profile_image_url
	  FROM users
	  WHERE users.username_lower = $1;`, username)
	err := row.Scan(&user.Name, &user.ProfileImage)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("profile data: %w", err)
	}
	return &user, nil
}

func (service *TweetService) LikeTweet(usernameLower string, tweetID int) (*TweetLike, error) {
	tweetLike := TweetLike{
		UsernameLower: usernameLower,
		TweetID:       tweetID,
	}
	// Insert into the DB the like data
	row := service.DB.QueryRow(`
	  INSERT INTO tweet_likes (tweet_id, user_username)
	  VALUES ($1, $2) RETURNING id;`, tweetLike.TweetID, tweetLike.UsernameLower)
	err := row.Scan(&tweetLike.ID)
	if err != nil {
		return nil, fmt.Errorf("create tweet like: %w", err)
	}
	return &tweetLike, nil
}

func (service *TweetService) GetTweetLikeCount(tweetID int) (int, error) {
	var likeCount int
	row := service.DB.QueryRow(`
        SELECT COUNT(*) FROM tweet_likes WHERE tweet_id = $1;
    `, tweetID)
	err := row.Scan(&likeCount)
	if err != nil {
		return 0, fmt.Errorf("get tweet like count: %w", err)
	}
	return likeCount, nil
}

func (service *TweetService) GetExistingLike(usernameLower string, tweetID int) (bool, error) {
	var liked bool
	row := service.DB.QueryRow(`
		SELECT EXISTS (SELECT 1 FROM tweet_likes WHERE tweet_id = $1 AND user_username = $2);`, tweetID, usernameLower)
	err := row.Scan(&liked)
	if err != nil {
		return liked, fmt.Errorf("get existing like: %w", err)
	}
	return liked, nil
}

func (service *TweetService) DislikeTweet(usernameLower string, tweetID int) error {
	_, err := service.DB.Exec(`
	  DELETE FROM tweet_likes
	  WHERE user_username = $1 AND tweet_id = $2;`, usernameLower, tweetID)
	if err != nil {
		return fmt.Errorf("dislike tweet: %w", err)
	}
	return nil
}

func (service *TweetService) GetTweetRepliesCount(tweetID int) (int, error) {
	var likeCount int
	row := service.DB.QueryRow(`
		SELECT COUNT(*) FROM tweets WHERE parent_tweet_id = $1;
    `, tweetID)
	err := row.Scan(&likeCount)
	if err != nil {
		return 0, fmt.Errorf("get tweet like count: %w", err)
	}
	return likeCount, nil
}

func (service *TweetService) CreateRetweet(usernameLower string, tweetID int) (*Retweet, error) {
	retweet := Retweet{
		UsernameLower: usernameLower,
		TweetID:       tweetID,
	}
	// Insert into the DB the like data
	row := service.DB.QueryRow(`
	  INSERT INTO retweets (tweet_id, user_username)
	  VALUES ($1, $2) RETURNING id;`, retweet.TweetID, retweet.UsernameLower)
	err := row.Scan(&retweet.ID)
	if err != nil {
		return nil, fmt.Errorf("create retweet: %w", err)
	}
	return &retweet, nil
}

func (service *TweetService) GetRetweetCount(tweetID int) (int, error) {
	var retweetCount int
	row := service.DB.QueryRow(`
        SELECT COUNT(*) FROM retweets WHERE tweet_id = $1;
    `, tweetID)
	err := row.Scan(&retweetCount)
	if err != nil {
		return 0, fmt.Errorf("get retweet count: %w", err)
	}
	return retweetCount, nil
}

func (service *TweetService) GetExistingRetweet(usernameLower string, tweetID int) (bool, error) {
	var retweeted bool
	row := service.DB.QueryRow(`
		SELECT EXISTS (SELECT 1 FROM retweets WHERE tweet_id = $1 AND user_username = $2);`, tweetID, usernameLower)
	err := row.Scan(&retweeted)
	if err != nil {
		return retweeted, fmt.Errorf("get existing like: %w", err)
	}
	return retweeted, nil
}

// func (service *TweetService) Edit(tweet *Tweet) error {
// 	// Text validation
// 	if len(tweet.Text) > 280 {
// 		return ErrLimitMaxText
// 	} else if len(tweet.Text) < 1 {
// 		return ErrLimitMinText
// 	}

// 	_, err := service.DB.Exec(`
// 	  UPDATE tweets
// 	  SET text = $2, created_at = $3
// 	  WHERE id = $1;`, tweet.ID, tweet.Text, tweet.CreatedAt)
// 	if err != nil {
// 		return fmt.Errorf("edit tweet: %w", err)
// 	}
// 	return nil
// }

func (service *TweetService) Delete(tweetID int) error {
	_, err := service.DB.Exec(`
	  DELETE FROM tweets
	  WHERE id = $1;`, tweetID)
	if err != nil {
		return fmt.Errorf("delete tweet by id: %w", err)
	}
	return nil
}

func (service *TweetService) UndoRetweet(usernameLower string, tweetID int) error {
	_, err := service.DB.Exec(`
	  DELETE FROM retweets
	  WHERE user_username = $1 AND tweet_id = $2;`, usernameLower, tweetID)
	if err != nil {
		return fmt.Errorf("undo retweet: %w", err)
	}
	return nil
}
