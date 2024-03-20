package controllers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"twitter-clone/errors"
	"twitter-clone/models"

	"github.com/go-chi/chi/v5"
)

type Image struct {
	ImageService *models.ImageService
}

func (i Image) RenderImages(w http.ResponseWriter, r *http.Request) {
	dir := chi.URLParam(r, "dir")
	filename := i.filename(r)

	image, err := i.ImageService.Image(dir, filename)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			http.Error(w, "Image not found", http.StatusNotFound)
			return
		}
		fmt.Println(err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	http.ServeFile(w, r, image.Path)
}

func (i Image) RenderTweetImages(w http.ResponseWriter, r *http.Request) {
	// /images/tweets/{tweetID}/photo-{idx}
	tweetID := chi.URLParam(r, "tweetID")
	filename := i.filename(r)

	imageTweetDir := filepath.Join("tweets", tweetID)
	image, err := i.ImageService.Image(imageTweetDir, filename)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			http.Error(w, "Image not found", http.StatusNotFound)
			return
		}
		fmt.Println(err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	http.ServeFile(w, r, image.Path)
}

func (i Image) RenderReplyImages(w http.ResponseWriter, r *http.Request) {
	// /images/replies/{tweetID}/{replyID}/photo-{idx}
	tweetID := chi.URLParam(r, "tweetID")
	replyID := chi.URLParam(r, "replyID")
	filename := i.filename(r)

	imageReplyDir := filepath.Join("replies", tweetID, replyID)
	image, err := i.ImageService.Image(imageReplyDir, filename)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			http.Error(w, "Image not found", http.StatusNotFound)
			return
		}
		fmt.Println(err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	http.ServeFile(w, r, image.Path)
}

func (i Image) filename(r *http.Request) string {
	filename := chi.URLParam(r, "filename")
	filename = filepath.Base(filename)
	return filename
}
