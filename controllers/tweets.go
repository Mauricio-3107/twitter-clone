package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"twitter-clone/context"
	"twitter-clone/errors"
	"twitter-clone/models"

	"github.com/go-chi/chi/v5"
)

type Tweet struct {
	Templates struct {
		Home        Template
		Errors      Template
		SingleTweet Template
		Profile     Template
	}
	TweetService    *models.TweetService
	UserService     *models.UserService
	ImageService    *models.ImageService
	BookmarkService *models.BookmarkService
}

// func (t Tweet) Home(w http.ResponseWriter, r *http.Request) {
// 	t.Templates.Home.Execute(w, r, nil)
// }

func (t Tweet) Create(w http.ResponseWriter, r *http.Request) {
	// check the images limit since if you don't do it, the text tweet will be creted anyway
	err := r.ParseMultipartForm(5 << 20) // 5mb
	if err != nil {
		http.Error(w, "Something went wrong here baby", http.StatusInternalServerError)
		return
	}
	fileHeaders := r.MultipartForm.File["images"]
	if len(fileHeaders) > 4 {
		err = models.ErrLimitMaxImages
		t.Templates.Errors.Execute(w, r, nil, err)
		return
	}
	// check if there are images so you handle the errors
	images := false
	if len(fileHeaders) > 0 {
		images = true
	}

	var data struct {
		UserID        int
		Text          string
		Username      string
		Name          string
		ProfileImage  string
		Images        bool
		ParentTweetID int
		QuotedTweetID int
	}
	// Retrieve the user
	data.Username = context.User(r.Context()).UsernameOriginal
	data.Name = context.User(r.Context()).Name
	data.ProfileImage = context.User(r.Context()).ProfileImage

	// Retrieve the values from the tweet form
	data.Text = r.FormValue("tweet")

	// Pass the info if there are images
	data.Images = images

	// Tweet or Reply
	parentTweetID, err := strconv.Atoi(r.FormValue("parentTweetID"))
	if err != nil {
		http.Error(w, "Invalid parent tweet ID", http.StatusNotFound)
		return
	}
	data.ParentTweetID = parentTweetID

	// Tweet or Quoted
	quotedTweetID, err := strconv.Atoi(r.FormValue("quotedTweetID"))
	if err != nil {
		http.Error(w, "Invalid parent tweet ID", http.StatusNotFound)
		return
	}
	data.QuotedTweetID = quotedTweetID

	// Pass these values to the db function so it is inserted
	tweet, err := t.TweetService.Create(data.Text, data.Username, data.Name, data.ProfileImage, data.Images, data.ParentTweetID, data.QuotedTweetID)
	if err != nil {
		if errors.Is(err, models.ErrLimitMaxText) {
			err = errors.Public(err, "Tweet no válido por exceso de caracteres (280 máximo)")
		} else if errors.Is(err, models.ErrEmptyTweet) {
			err = errors.Public(err, "Debes ingresar al menos 1 caracter o 1 imagen")
		}
		t.Templates.Errors.Execute(w, r, nil, err)
		return
	}

	// Retrieve the images from the form and create in the hard disk and their URLs in the DB
	for idx, fileHeader := range fileHeaders {
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
		defer file.Close()
		err = t.ImageService.CreateTweetImages(tweet.ID, idx+1, fileHeader.Filename, file)
		if err != nil {
			errDelete := t.TweetService.Delete(tweet.ID)
			if errDelete != nil {
				http.Error(w, "Something went wrong", http.StatusInternalServerError)
			}
			var fileErr models.FileError
			if errors.As(err, &fileErr) {
				fmt.Println(fileErr.Issue)
				msg := fmt.Sprintf("%v has an invalid content type or extension (%v). Only png, gif, and jpg files can be uploaded.", fileHeader.Filename, fileErr.Issue)
				http.Error(w, msg, http.StatusBadRequest)
				return
			}
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}
	// Now save the path to the DB. Ask to the hard disk for the images so you make sure that they have been saved
	// /images/tweets/{tweetID}/photo-{idx}
	var imagesURL []string
	imageTweetDir := filepath.Join("tweets", strconv.Itoa(tweet.ID))
	for i := 0; i < len(fileHeaders); i++ {
		formattedFilename := fmt.Sprintf("photo-%d.jpeg", i+1)
		image, err := t.ImageService.Image(imageTweetDir, formattedFilename)
		if err != nil {
			http.Error(w, "Something went wrong queriyng image tweet", http.StatusInternalServerError)
			return
		}
		tweetImageURL := formatSlashes(image.Path)
		imagesURL = append(imagesURL, tweetImageURL)
	}
	tweet, err = t.TweetService.CreateTweetImages(*tweet, imagesURL...)
	if err != nil {
		t.Templates.Errors.Execute(w, r, nil, err)
		return
	}

	// Not doing anything when retrieving the tweet
	_ = tweet
	// Redirect to home or to the tweet ID
	if parentTweetID != 0 {
		// /{username}/status/{tweetID}
		usernameTweet := r.FormValue("usernameTweet")
		editPath := fmt.Sprintf("/%s/status/%d", usernameTweet, parentTweetID)
		http.Redirect(w, r, editPath, http.StatusFound)
	} else {
		http.Redirect(w, r, "/home", http.StatusFound)
	}
}

func (t Tweet) CreateAjax(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	err := r.ParseMultipartForm(5 << 20) // 10 MB max size
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	fileHeaders := r.MultipartForm.File["images"]
	if len(fileHeaders) > 4 {
		err = models.ErrLimitMaxImages
		t.Templates.Errors.Execute(w, r, nil, err)
		return
	}
	// check if there are images so you handle the errors
	images := false
	if len(fileHeaders) > 0 {
		images = true
	}

	var data struct {
		UserID        int
		Text          string
		Username      string
		Name          string
		ProfileImage  string
		Images        bool
		ParentTweetID int
		QuotedTweetID int
	}

	// Retrieve the user
	data.Username = context.User(r.Context()).UsernameOriginal
	data.Name = context.User(r.Context()).Name
	data.ProfileImage = context.User(r.Context()).ProfileImage

	// Extract form values
	data.Text = r.FormValue("tweet")

	// Pass the info if there are images
	data.Images = images

	// Tweet or Reply
	parentTweetID, err := strconv.Atoi(r.FormValue("parentTweetID"))
	if err != nil {
		http.Error(w, "Invalid parent tweet ID", http.StatusBadRequest)
		return
	}
	data.ParentTweetID = parentTweetID

	// Tweet or Quoted
	quotedTweetID, err := strconv.Atoi(r.FormValue("quotedTweetID"))
	if err != nil {
		http.Error(w, "Invalid parent tweet ID", http.StatusNotFound)
		return
	}
	data.QuotedTweetID = quotedTweetID

	// Pass these values to the db function so it is inserted
	tweet, err := t.TweetService.Create(data.Text, data.Username, data.Name, data.ProfileImage, data.Images, data.ParentTweetID, data.QuotedTweetID)
	if err != nil {
		if errors.Is(err, models.ErrLimitMaxText) {
			err = errors.Public(err, "Tweet no válido por exceso de caracteres (280 máximo)")
		} else if errors.Is(err, models.ErrEmptyTweet) {
			err = errors.Public(err, "Debes ingresar al menos 1 caracter o una images")
		}
		t.Templates.Errors.Execute(w, r, nil, err)
		return
	}

	// Retrieve the images from the form and create in the hard disk and their URLs in the DB
	for idx, fileHeader := range fileHeaders {
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
		defer file.Close()
		err = t.ImageService.CreateTweetImages(tweet.ID, idx+1, fileHeader.Filename, file)
		if err != nil {
			var fileErr models.FileError
			if errors.As(err, &fileErr) {
				msg := fmt.Sprintf("%v has an invalid content type or extension. Only png, gif, and jpg files can be uploaded.", fileHeader.Filename)
				http.Error(w, msg, http.StatusBadRequest)
				return
			}
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}
	// Now save the path to the DB. Ask to the hard disk for the images so you make sure that they have been saved
	// /images/tweets/{tweetID}/photo-{idx}
	var imagesURL []string
	imageTweetDir := filepath.Join("tweets", strconv.Itoa(tweet.ID))
	for i := 0; i < len(fileHeaders); i++ {
		formattedFilename := fmt.Sprintf("photo-%d.jpeg", i+1)
		image, err := t.ImageService.Image(imageTweetDir, formattedFilename)
		if err != nil {
			http.Error(w, "Something went wrong queriyng image tweet", http.StatusInternalServerError)
			return
		}
		tweetImageURL := formatSlashes(image.Path)
		imagesURL = append(imagesURL, tweetImageURL)
	}
	tweet, err = t.TweetService.CreateTweetImages(*tweet, imagesURL...)
	if err != nil {
		t.Templates.Errors.Execute(w, r, nil, err)
		return
	}
	_ = tweet

	// Tweet or reply
	// Count replies
	countReplies, err := t.TweetService.GetTweetRepliesCount(parentTweetID)
	if err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	// Return JSON response indicating success
	w.Header().Set("Content-Type", "application/json")
	if parentTweetID != 0 {
		response := struct {
			Success      bool `json:"success"`
			RepliesCount int  `json:"repliesCount"`
		}{
			Success:      true,
			RepliesCount: countReplies,
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

func (t Tweet) TweetsByUserFollowing(w http.ResponseWriter, r *http.Request) {
	// Retrieve the name and username from context
	name := context.User(r.Context()).Name
	usernameCtx := context.User(r.Context()).UsernameOriginal
	// Retrieve all their tweets
	tweets, tweetsIDs, err := t.TweetService.FeedWithUserTweets(usernameCtx)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}
	// Curate the data
	type quotedTweetData struct {
		TweetID       int
		UserID        int
		Text          string
		CreatedAt     string
		Username      string
		Name          string
		ProfileImage  string
		ImagesURL     []string
		ParentTweetID int
		QuotedTweetID int
		CountReplies  int
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
		QuotedTweet   quotedTweetData
	}
	var data struct {
		TweetsData []tweetData
		TweetIDs   []int
		Name       string
		Username   string
	}
	data.Name = name
	data.Username = usernameCtx
	for _, tweet := range tweets {
		td := tweetData{
			TweetID:       tweet.ID,
			UserID:        tweet.ID,
			Text:          tweet.Text,
			Username:      tweet.UsernameOriginal,
			Name:          tweet.Name,
			CreatedAt:     formatTimeAgo(tweet.CreatedAt),
			ProfileImage:  tweet.ProfileImage,
			Retweeted:     tweet.Retweeted,
			ImagesURL:     tweet.ImagesURL,
			QuotedTweetID: tweet.QuotedTweetID,
		}
		var quotedTweet *models.Tweet
		if tweet.QuotedTweetID != 0 {
			quotedTweet, _, err = t.TweetService.ByTweetID(tweet.QuotedTweetID)
			if err != nil {
				if errors.Is(err, models.ErrNotFound) {
					http.Error(w, "Tweet not found", http.StatusNotFound)
					return
				}
				fmt.Println(err)
				http.Error(w, "Something went wrong", http.StatusInternalServerError)
				return
			}
			td.QuotedTweet = quotedTweetData{
				TweetID:       quotedTweet.ID,
				Text:          quotedTweet.Text,
				Name:          quotedTweet.Name,
				CreatedAt:     quotedTweet.CreatedAt.Format("3:04 PM · Jan 2, 2006"),
				Username:      quotedTweet.UsernameOriginal,
				ProfileImage:  quotedTweet.ProfileImage,
				ImagesURL:     quotedTweet.ImagesURL,
				ParentTweetID: quotedTweet.ParentTweetID,
				QuotedTweetID: quotedTweet.QuotedTweetID,
			}
		} else {
			_ = quotedTweet
		}

		data.TweetsData = append(data.TweetsData, td)

	}

	data.TweetIDs = append(data.TweetIDs, tweetsIDs...)

	// Pass the data into the home template and render it
	t.Templates.Home.Execute(w, r, data)
}

func (t Tweet) Profile(w http.ResponseWriter, r *http.Request) {
	isOwnProfile := false
	isFollowing := false
	// Retrieve the username param
	usernameParam := chi.URLParam(r, "username")

	// Try to retrieve the context user
	usernameCtx := context.User(r.Context()).UsernameOriginal
	if usernameParam == usernameCtx {
		isOwnProfile = true
	}

	if !isOwnProfile {
		// You have to read if there is a following and pass it. In the gohtml you can insert this info inside the else isOwnProfile
		checkFollowing, err := t.UserService.CheckFollowing(usernameCtx, usernameParam)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Something went wrong isFollowing", http.StatusInternalServerError)
			return
		}
		isFollowing = checkFollowing
	}

	// Retrieve profile data
	user, err := t.TweetService.ProfileData(usernameParam)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			http.Error(w, "The requested username doesn't exist", http.StatusNotFound)
			return
		}
		fmt.Println(err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	// Retrieve Following and Followers quantities
	followingCount, err := t.UserService.GetFollowingCount(usernameParam)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}
	followerCount, err := t.UserService.GetFollowerCount(usernameParam)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}

	// Retrieve all their tweets
	tweets, tweetsIDs, err := t.TweetService.ByUserNameOriginal(usernameParam)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}

	// Curate the data
	type quotedTweetData struct {
		TweetID       int
		UserID        int
		Text          string
		CreatedAt     string
		Username      string
		Name          string
		ProfileImage  string
		ImagesURL     []string
		ParentTweetID int
		QuotedTweetID int
		CountReplies  int
	}
	type tweetData struct {
		TweetID       int
		UserID        int
		Text          string
		CreatedAt     string
		Username      string
		Name          string
		ProfileImage  string
		ImagesURL     []string
		Retweeted     bool
		QuotedTweetID int
		QuotedTweet   quotedTweetData
	}
	type profile struct {
		Username        string
		Name            string
		ProfileImage    string
		IsOwnProfile    bool
		IsFollowing     bool
		FollowingsCount int
		FollowersCount  int
	}
	var data struct {
		TweetsData  []tweetData
		ProfileData profile
		TweetIDs    []int
	}
	data.ProfileData = profile{
		Username:        user.UsernameOriginal,
		Name:            user.Name,
		ProfileImage:    user.ProfileImage,
		IsOwnProfile:    isOwnProfile,
		IsFollowing:     isFollowing,
		FollowingsCount: followingCount,
		FollowersCount:  followerCount,
	}
	for _, tweet := range tweets {
		td := tweetData{
			TweetID:       tweet.ID,
			UserID:        tweet.ID,
			Text:          tweet.Text,
			Username:      tweet.UsernameOriginal,
			Name:          tweet.Name,
			CreatedAt:     formatTimeAgo(tweet.CreatedAt),
			ProfileImage:  tweet.ProfileImage,
			ImagesURL:     tweet.ImagesURL,
			Retweeted:     tweet.Retweeted,
			QuotedTweetID: tweet.QuotedTweetID,
		}
		var quotedTweet *models.Tweet
		if tweet.QuotedTweetID != 0 {
			quotedTweet, _, err = t.TweetService.ByTweetID(tweet.QuotedTweetID)
			if err != nil {
				if errors.Is(err, models.ErrNotFound) {
					http.Error(w, "Tweet not found", http.StatusNotFound)
					return
				}
				fmt.Println(err)
				http.Error(w, "Something went wrong", http.StatusInternalServerError)
				return
			}
			td.QuotedTweet = quotedTweetData{
				TweetID:       quotedTweet.ID,
				Text:          quotedTweet.Text,
				Name:          quotedTweet.Name,
				CreatedAt:     quotedTweet.CreatedAt.Format("3:04 PM · Jan 2, 2006"),
				Username:      quotedTweet.UsernameOriginal,
				ProfileImage:  quotedTweet.ProfileImage,
				ImagesURL:     quotedTweet.ImagesURL,
				ParentTweetID: quotedTweet.ParentTweetID,
				QuotedTweetID: quotedTweet.QuotedTweetID,
			}
		} else {
			_ = quotedTweet
		}
		data.TweetsData = append(data.TweetsData, td)
	}

	data.TweetIDs = append(data.TweetIDs, tweetsIDs...)

	// Pass the data into the home template and render it
	t.Templates.Profile.Execute(w, r, data)
}

func (t Tweet) RenderSingleTweet(w http.ResponseWriter, r *http.Request) {
	// Grab the username and the tweet ID
	usernameParam := chi.URLParam(r, "username")
	usernameLower := strings.ToLower(usernameParam)
	tweetID, err := strconv.Atoi(chi.URLParam(r, "tweetID"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusNotFound)
		return
	}
	// Query for the tweet
	tweet, replies, err := t.TweetService.ByTweetID(tweetID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			http.Error(w, "Tweet not found", http.StatusNotFound)
			return
		}
		fmt.Println(err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	// Verify that the found tweet belong to the username parameter
	if tweet.UsernameOriginal != usernameParam {
		http.Error(w, "The tweet you are looking does not belong to the requested username", http.StatusBadRequest)
		return
	}
	// Verify if quoted tweet
	var quotedTweet *models.Tweet
	if tweet.QuotedTweetID != 0 {
		quotedTweet, _, err = t.TweetService.ByTweetID(tweet.QuotedTweetID)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				http.Error(w, "Tweet not found", http.StatusNotFound)
				return
			}
			fmt.Println(err)
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	} else {
		_ = quotedTweet
	}

	// Count replies
	countReplies, err := t.TweetService.GetTweetRepliesCount(tweetID)
	if err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	isRetweeted, err := t.TweetService.GetExistingRetweet(usernameLower, tweetID)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to check existing retweet", http.StatusInternalServerError)
		return
	}
	fmt.Println("is retweet:", isRetweeted)

	// Curate the data
	type quotedTweetData struct {
		TweetID       int
		UserID        int
		Text          string
		CreatedAt     string
		Username      string
		Name          string
		ProfileImage  string
		ImagesURL     []string
		ParentTweetID int
		QuotedTweetID int
		CountReplies  int
	}
	type tweetData struct {
		TweetID       int
		UserID        int
		Text          string
		CreatedAt     string
		Username      string
		Name          string
		ProfileImage  string
		ImagesURL     []string
		ParentTweetID int
		QuotedTweetID int
		CountReplies  int
		QuotedTweet   quotedTweetData
		IsRetweeted   bool
	}
	var data struct {
		TweetData tweetData
		Replies   []tweetData
		TweetIDs  []int
	}

	data.TweetData = tweetData{
		TweetID:       tweetID,
		Text:          tweet.Text,
		Name:          tweet.Name,
		CreatedAt:     tweet.CreatedAt.Format("3:04 PM · Jan 2, 2006"),
		Username:      tweet.UsernameOriginal,
		ProfileImage:  tweet.ProfileImage,
		ImagesURL:     tweet.ImagesURL,
		ParentTweetID: tweet.ParentTweetID,
		QuotedTweetID: tweet.QuotedTweetID,
		IsRetweeted:   isRetweeted,
	}

	if tweet.QuotedTweetID != 0 {
		data.TweetData.QuotedTweet = quotedTweetData{
			TweetID:       quotedTweet.ID,
			Text:          quotedTweet.Text,
			Name:          quotedTweet.Name,
			CreatedAt:     quotedTweet.CreatedAt.Format("3:04 PM · Jan 2, 2006"),
			Username:      quotedTweet.UsernameOriginal,
			ProfileImage:  quotedTweet.ProfileImage,
			ImagesURL:     quotedTweet.ImagesURL,
			ParentTweetID: quotedTweet.ParentTweetID,
			QuotedTweetID: quotedTweet.QuotedTweetID,
		}
	}

	for _, replyTweet := range replies {
		data.Replies = append(data.Replies, tweetData{
			TweetID:       replyTweet.ID,
			Text:          replyTweet.Text,
			Name:          replyTweet.Name,
			CreatedAt:     replyTweet.CreatedAt.Format("3:04 PM · Jan 2, 2006"),
			Username:      replyTweet.UsernameOriginal,
			ProfileImage:  replyTweet.ProfileImage,
			ImagesURL:     replyTweet.ImagesURL,
			ParentTweetID: replyTweet.ParentTweetID,
		})
		// Append the replies tweetIDs to the TweetIDs slice
		data.TweetIDs = append(data.TweetIDs, replyTweet.ID)
	}
	// Append the tweetID to the TweetIDs slice
	data.TweetIDs = append(data.TweetIDs, tweetID)

	// Count replies added to data
	data.TweetData.CountReplies = countReplies

	// Render the post template with the data
	t.Templates.SingleTweet.Execute(w, r, data)
}

func (t Tweet) HandleLikeDislikeTweet(w http.ResponseWriter, r *http.Request) {
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

	existingLike, err := t.TweetService.GetExistingLike(usernameLower, requestData.TweetID)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to check existing like", http.StatusInternalServerError)
		return
	}
	if existingLike {
		// Dislike the tweet
		err = t.TweetService.DislikeTweet(usernameLower, requestData.TweetID)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Disliking tweet", http.StatusInternalServerError)
			return
		}
	} else {
		// Like the tweet
		_, err := t.TweetService.LikeTweet(usernameLower, requestData.TweetID)
		if err != nil {
			http.Error(w, "Liking tweet", http.StatusInternalServerError)
			return
		}
	}

	// Get updated like count
	likeCount, err := t.TweetService.GetTweetLikeCount(requestData.TweetID)
	if err != nil {
		http.Error(w, "Failed to get updated like count", http.StatusInternalServerError)
		return
	}

	// isRetweeted
	isRetweeted, err := t.TweetService.GetExistingRetweet(usernameLower, requestData.TweetID)
	if err != nil {
		http.Error(w, "Failed to get updated like count", http.StatusInternalServerError)
		return
	}

	// Return updated like count as JSON response
	w.Header().Set("Content-Type", "application/json")
	response := struct {
		LikeCount   int  `json:"likeCount"`
		IsRetweeted bool `json:"isRetweeted"`
	}{
		LikeCount:   likeCount,
		IsRetweeted: isRetweeted,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (t Tweet) GetTweetDataCountHandler(w http.ResponseWriter, r *http.Request) {
	// Normalize the username
	usernameOriginal := context.User(r.Context()).UsernameOriginal
	usernameLower := strings.ToLower(usernameOriginal)

	// Parse the tweet ID from the request paramters
	tweetIDStr := r.URL.Query().Get("tweetID")
	tweetID, err := strconv.Atoi(tweetIDStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusNotFound)
		return
	}
	// Query the database to get the like count for the tweet with the specified ID
	// (Assuming you have a function like GetTweetLikeCountFromDB(tweetID) to retrieve the like count)
	likeCount, err := t.TweetService.GetTweetLikeCount(tweetID)
	if err != nil {
		http.Error(w, "Failed to get tweet like count", http.StatusInternalServerError)
		return
	}
	existingLike, err := t.TweetService.GetExistingLike(usernameLower, tweetID)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to check existing like", http.StatusInternalServerError)
		return
	}

	// retweets
	retweetsCount, err := t.TweetService.GetRetweetCount(tweetID)
	if err != nil {
		http.Error(w, "Failed to get retweets count", http.StatusInternalServerError)
		return
	}
	isRetweeted, err := t.TweetService.GetExistingRetweet(usernameLower, tweetID)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to check existing retweet", http.StatusInternalServerError)
		return
	}

	// replies
	// Count replies
	countReplies, err := t.TweetService.GetTweetRepliesCount(tweetID)
	if err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	// Bookmarked
	isBookmarked, err := t.BookmarkService.GetExistingBookmark(tweetID, usernameLower)
	if err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	// // Return the like count as a JSON response
	// json.NewEncoder(w).Encode(map[string]int{"likeCount": likeCount, "existingLike": existingLike})
	// Return updated like count as JSON response
	w.Header().Set("Content-Type", "application/json")
	response := struct {
		LikeCount     int  `json:"likeCount"`
		ExistingLike  bool `json:"existingLike"`
		RetweetsCount int  `json:"retweetsCount"`
		IsRetweeted   bool `json:"isRetweeted"`
		RepliesCount  int  `json:"repliesCount"`
		IsBookmarked  bool `json:"isBookmarked"`
	}{
		LikeCount:     likeCount,
		ExistingLike:  existingLike,
		RetweetsCount: retweetsCount,
		IsRetweeted:   isRetweeted,
		RepliesCount:  countReplies,
		IsBookmarked:  isBookmarked,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (t Tweet) HandleRetweet(w http.ResponseWriter, r *http.Request) {
	// Normalize the username
	usernameOriginal := context.User(r.Context()).UsernameOriginal
	usernameLower := strings.ToLower(usernameOriginal)

	var requestData struct {
		TweetID int `json:"tweetID"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}
	isRetweeted, err := t.TweetService.GetExistingRetweet(usernameLower, requestData.TweetID)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to check existing like", http.StatusInternalServerError)
		return
	}
	if isRetweeted {
		// Undo the retweet
		err = t.TweetService.UndoRetweet(usernameLower, requestData.TweetID)
		if err != nil {
			http.Error(w, "Failed to create retweet", http.StatusInternalServerError)
			return
		}
	} else {
		_, err := t.TweetService.CreateRetweet(usernameLower, requestData.TweetID)
		if err != nil {
			http.Error(w, "Failed to create retweet", http.StatusInternalServerError)
			return
		}
	}

	// Get retweets count from a tweet
	retweetsCount, err := t.TweetService.GetRetweetCount(requestData.TweetID)
	if err != nil {
		http.Error(w, "Failed to get retweets count", http.StatusInternalServerError)
		return
	}
	// Return updated like count as JSON response
	w.Header().Set("Content-Type", "application/json")
	response := struct {
		RetweetsCount int  `json:"retweetsCount"`
		IsRetweeted   bool `json:"isRetweeted"`
	}{
		RetweetsCount: retweetsCount,
		IsRetweeted:   isRetweeted,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func formatTimeAgo(createdAt time.Time) string {
	duration := time.Since(createdAt)
	now := time.Now()
	switch {
	case duration.Seconds() < 60:
		return "just now"
	case duration.Minutes() < 60:
		return fmt.Sprintf("%.0fm", duration.Minutes())
	case duration.Hours() < 24:
		return fmt.Sprintf("%.0fh", duration.Hours())
	default:
		if now.Year() == createdAt.Year() {
			// Format for the present year
			return createdAt.Format("Jan 2")
		} else {
			// Format for past years
			return createdAt.Format("Jan 2, 2006")
		}
	}
}
