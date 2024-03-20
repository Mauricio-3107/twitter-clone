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
WHERE
    tweets.user_username = 'mauri3107'
    OR tweets.user_username IN (
        SELECT
            following.following_username
        FROM
            following
        WHERE
            following.user_username = 'mauri3107'
    )
GROUP BY
    tweets.id,
    users.username_original,
    users.name,
    users.profile_image_url
ORDER BY
    tweets.created_at DESC;

-- tweetsByUsername
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
    users.username_original = $ 1
GROUP BY
    tweets.id,
    users.name,
    users.profile_image_url
ORDER BY
    tweets.created_at DESC;

-- tweet replies
SELECT
    tweet_replies.id,
    tweet_replies.text,
    tweet_replies.created_at,
    users.name,
    users.profile_image_url,
    users.username_original,
    ARRAY_AGG(reply_images.image_url)
FROM
    tweet_replies
    JOIN users ON tweet_replies.user_username = users.username_lower
    LEFT JOIN reply_images ON tweet_replies.id = reply_images.reply_id
WHERE
    tweet_replies.tweet_id = 1
GROUP BY
    tweet_replies.id,
    users.name,
    users.profile_image_url,
    users.username_original
ORDER BY
    tweet_replies.created_at DESC;

-- Join tweets and retweets
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
    users.username_original = $ 1
    AND tweets.parent_tweet_id = 0
GROUP BY
    tweets.id,
    users.name,
    users.profile_image_url
UNION
ALL
SELECT
    tweets.id,
    tweets.text,
    tweets.created_at,
    users.name,
    users.profile_image_url,
    ARRAY_AGG(tweet_images.image_url)
FROM
    retweets
    JOIN tweets ON retweets.tweet_id = tweets.id
    JOIN users ON tweets.user_username = users.username_lower
    LEFT JOIN tweet_images ON tweets.id = tweet_images.tweet_id
WHERE
    retweets.user_username = $ 1
    AND tweets.parent_tweet_id = 0
GROUP BY
    tweets.id,
    users.name,
    users.profile_image_url
ORDER BY
    created_at DESC;

-- Feed with retweets
SELECT
    tweets.id,
    tweets.text,
    tweets.created_at,
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
        tweets.user_username = $ 1
        OR tweets.user_username IN (
            SELECT
                following.following_username
            FROM
                following
            WHERE
                following.user_username = $ 1
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
    retweets.user_username = $ 1
    AND tweets.parent_tweet_id = 0
GROUP BY
    tweets.id,
    users.username_original,
    users.name,
    users.profile_image_url
ORDER BY
    created_at DESC;

-- Bookmarks
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
    JOIN users ON tweets.user_username = users.username_lower
    LEFT JOIN tweet_images ON tweets.id = tweet_images.tweet_id
    JOIN bookmarks ON tweets.id = bookmarks.tweet_id
WHERE
    bookmarks.user_username = $ 1
    AND tweets.parent_tweet_id = 0
GROUP BY
    tweets.id,
    users.name,
    users.profile_image_url,
    users.username_original
ORDER BY
    tweets.created_at DESC;