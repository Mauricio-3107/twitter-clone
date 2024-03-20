package models

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

type Image struct {
	Dir      string
	Path     string
	Filename string
}

type ImageService struct {
	// ImagesDir is used to tell the TweetService where to store and locate images. If not set, the GalleryService will default to using the "images" directory.
	ImagesDir string
}

// Images
func (service *ImageService) imageDir(dir string) string {
	imgDir := service.ImagesDir
	if imgDir == "" {
		imgDir = "images"
	}
	return filepath.Join(imgDir, dir)
}

func (service *ImageService) ImageTweetDir(tweetID int) string {
	// /images/tweets
	imgDir := service.imageDir("tweets")
	tweetIDStr := strconv.Itoa(tweetID)
	// /images/tweets/tweetID
	return filepath.Join(imgDir, tweetIDStr)
}

func (service *ImageService) ImageReplyDir(tweetID, replyID int) string {
	// /images/tweets
	imgDir := service.imageDir("replies")
	tweetIDStr := strconv.Itoa(tweetID)
	replyIDStr := strconv.Itoa(replyID)
	// /images/replies/{tweetID}/{replyID}
	return filepath.Join(imgDir, tweetIDStr, replyIDStr)
}

func (service *ImageService) Image(dir, filename string) (Image, error) {
	imagePath := filepath.Join(service.imageDir(dir), filename)
	_, err := os.Stat(imagePath)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Image{}, ErrNotFound
		}
		return Image{}, fmt.Errorf("querying image: %w", err)
	}
	return Image{
		Filename: filename,
		Path:     imagePath,
		Dir:      dir,
	}, nil
}

func (service *ImageService) CreateProfileImage(username, filename string, contents io.ReadSeeker) error {
	err := checkContentType(contents, service.imageContentTypes())
	if err != nil {
		return fmt.Errorf("creating profile image %v: %w", filename, err)
	}
	err = checkExtension(filename, service.extensions())
	if err != nil {
		return fmt.Errorf("creating profile image %v: %w", filename, err)
	}

	imgDir := service.imageDir("users")
	err = os.MkdirAll(imgDir, 0755)
	if err != nil {
		return fmt.Errorf("creating users directory: %w", err)
	}
	formattedFilename := fmt.Sprintf("profile-img-%s.jpeg", username)
	imagePath := filepath.Join(imgDir, formattedFilename)
	dst, err := os.Create(imagePath)
	if err != nil {
		return fmt.Errorf("creating profile image file: %w", err)
	}
	defer dst.Close()

	_, err = io.Copy(dst, contents)
	if err != nil {
		return fmt.Errorf("copying contents to profile image: %w", err)
	}
	return nil
}

func (service *ImageService) CreateTweetImages(tweetID, numImg int, filename string, contents io.ReadSeeker) error {
	err := checkContentType(contents, service.imageContentTypes())
	if err != nil {
		return fmt.Errorf("creating tweet images %v: %w", filename, err)
	}
	err = checkExtension(filename, service.extensions())
	if err != nil {
		return fmt.Errorf("creating tweet images %v: %w", filename, err)
	}

	imageTweetDir := service.ImageTweetDir(tweetID)
	err = os.MkdirAll(imageTweetDir, 0755)
	if err != nil {
		return fmt.Errorf("creating tweet images directory: %w", err)
	}
	
	numImgStr := strconv.Itoa(numImg)
	formattedFilename := fmt.Sprintf("photo-%s.jpeg", numImgStr)
	imagePath := filepath.Join(imageTweetDir, formattedFilename)
	dst, err := os.Create(imagePath)
	if err != nil {
		return fmt.Errorf("creating tweet images file: %w", err)
	}
	defer dst.Close()

	_, err = io.Copy(dst, contents)
	if err != nil {
		return fmt.Errorf("copying contents to tweet images: %w", err)
	}
	return nil
}

func (service *ImageService) CreateReplyImages(tweetID, numImg, replyID int, filename string, contents io.ReadSeeker) error {
	err := checkContentType(contents, service.imageContentTypes())
	if err != nil {
		return fmt.Errorf("creating tweet images %v: %w", filename, err)
	}
	err = checkExtension(filename, service.extensions())
	if err != nil {
		return fmt.Errorf("creating tweet images %v: %w", filename, err)
	}

	imageReplyDir := service.ImageReplyDir(tweetID, replyID)
	err = os.MkdirAll(imageReplyDir, 0755)
	if err != nil {
		return fmt.Errorf("creating tweet images directory: %w", err)
	}

	numImgStr := strconv.Itoa(numImg)
	formattedFilename := fmt.Sprintf("photo-%s.jpeg", numImgStr)
	imagePath := filepath.Join(imageReplyDir, formattedFilename)
	dst, err := os.Create(imagePath)
	if err != nil {
		return fmt.Errorf("creating tweet images file: %w", err)
	}
	defer dst.Close()

	_, err = io.Copy(dst, contents)
	if err != nil {
		return fmt.Errorf("copying contents to tweet images: %w", err)
	}
	return nil
}

func (service *ImageService) extensions() []string {
	return []string{".png", ".jpg", ".jpeg"}
}

func (service *ImageService) imageContentTypes() []string {
	return []string{"image/png", "image/jpg", "image/jpeg"}
}
