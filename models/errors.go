package models

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

var (
	ErrNotFound       = errors.New("models: resource could not be found")
	ErrEmailTaken     = errors.New("models: email address is already in use")
	ErrUsernameTaken  = errors.New("models: username is already in use")
	ErrLimitMaxText   = errors.New("models: max limit text is 280 characters long")
	ErrLimitMinText   = errors.New("models: min limit text is 1 character long")
	ErrEmptyTweet     = errors.New("models: empty tweet")
	ErrLimitMaxImages = errors.New("models: max limit images on a tweet is 4")
)

type FileError struct {
	Issue string
}

func (fe FileError) Error() string {
	return fmt.Sprintf("invalid file: %s", fe.Issue)
}

func checkContentType(r io.ReadSeeker, allowedTypes []string) error {
	testBytes := make([]byte, 512)
	_, err := r.Read(testBytes)
	if err != nil {
		return fmt.Errorf("checking content type: %w", err)
	}
	_, err = r.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("checking content type: %w", err)
	}
	contentType := http.DetectContentType(testBytes)
	for _, t := range allowedTypes {
		if contentType == t {
			return nil
		}
	}
	return FileError{
		Issue: fmt.Sprintf("invalid content type: %v", contentType),
	}
}

func checkExtension(filename string, allowedExtensions []string) error {
	if hasExtension(filename, allowedExtensions) {
		return nil
	}
	return FileError{
		Issue: fmt.Sprintf("invalid extension: %v", filepath.Ext(filename)),
	}
}

func hasExtension(file string, extensions []string) bool {
	for _, ext := range extensions {
		file = strings.ToLower(file)
		ext = strings.ToLower(ext)
		if filepath.Ext(file) == ext {
			return true
		}
	}
	return false
}
