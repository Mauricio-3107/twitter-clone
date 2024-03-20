package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"twitter-clone/context"
	"twitter-clone/models"
)

type Bookmark struct {
	Templates struct {
		Bookmark Template
	}
	BookmarkService *models.BookmarkService
	TweetService    *models.TweetService
}

func (b Bookmark) Create(w http.ResponseWriter, r *http.Request) {
	// Normalize the username
	usernameOriginal := context.User(r.Context()).UsernameOriginal
	usernameLower := strings.ToLower(usernameOriginal)

	// Parse JSON request body
	var requestData struct {
		TweetID int `json:"tweetID"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}

	isBookmarked, err := b.BookmarkService.GetExistingBookmark(requestData.TweetID, usernameLower)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to get existing bookmark", http.StatusInternalServerError)
		return
	}

	if isBookmarked {
		err := b.BookmarkService.Delete(requestData.TweetID, usernameLower)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Failed to delete bookmark", http.StatusInternalServerError)
			return
		}
	} else {
		_, err := b.BookmarkService.Create(requestData.TweetID, usernameLower)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Failed to create bookmark", http.StatusInternalServerError)
			return
		}
	}
}

func (b Bookmark) RenderBookmarks(w http.ResponseWriter, r *http.Request) {
	usernameCtx := context.User(r.Context()).UsernameOriginal
	usernameLower := strings.ToLower(usernameCtx)
	tweets, tweetsIDs, err := b.BookmarkService.ByUserNameLower(usernameLower)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}
	type tweetData struct {
		TweetID       int
		UserID        int
		Text          string
		CreatedAt     string
		Username      string
		Name          string
		ProfileImage  string
		Retweeted     bool
		ImagesURL     []string
		QuotedTweetID int
	}
	var data struct {
		TweetsData []tweetData
		TweetIDs   []int
		Name       string
		Username   string
	}
	for _, tweet := range tweets {
		td := tweetData{
			TweetID:      tweet.ID,
			UserID:       tweet.ID,
			Text:         tweet.Text,
			Username:     tweet.UsernameOriginal,
			Name:         tweet.Name,
			CreatedAt:    formatTimeAgo(tweet.CreatedAt),
			ProfileImage: tweet.ProfileImage,
			Retweeted:    tweet.Retweeted,
			ImagesURL:    tweet.ImagesURL,
		}
		data.TweetsData = append(data.TweetsData, td)
	}
	data.TweetIDs = append(data.TweetIDs, tweetsIDs...)
	data.Username = usernameCtx
	data.Name = context.User(r.Context()).Name
	b.Templates.Bookmark.Execute(w, r, data)
}
