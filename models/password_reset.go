package models

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
	"twitter-clone/rand"
)

const (
	// DefaultResetDuration is the default time that a PasswordReset is
	// valid for.
	DefaultResetDuration = 1 * time.Hour
)

type PasswordReset struct {
	ID     int
	UserID int
	// Token is inly set when creating a new session. When we loop up s session, this will be left empty, as we only store the hash of a session token in our DB and we cannot reverse it into a raw token
	Token     string
	TokenHash string
	ExpiredAt time.Time
}

type PasswordResetService struct {
	DB *sql.DB
	// BytesPerToken is used to determine how many bytes to use when generating
	// each password reset token. If this value is not set or is less than the
	// MinBytesPerToken const it will be ignored and MinBytesPerToken will be
	// used.
	BytesPerToken int
	// Duration is the amount of time that a PasswordReset is valid for. Defaults to DefaultResetDuration
	Duration time.Duration
}

func (service *PasswordResetService) Create(email string) (*PasswordReset, error) {
	// Verify we have a valid email address for a user
	email = strings.ToLower(email)
	var userID int
	row := service.DB.QueryRow(`
		SELECT id FROM users WHERE email = $1;`, email)
	err := row.Scan(&userID)
	if err != nil {
		// Verify we have a valid email address for a user
		return nil, fmt.Errorf("create: %w", err)
	}

	// Build the PasswordReset
	bytesPerToken := service.BytesPerToken
	if bytesPerToken < MinBytesPerToken {
		bytesPerToken = MinBytesPerToken
	}
	token, err := rand.String(bytesPerToken)
	if err != nil {
		return nil, fmt.Errorf("create: %w", err)
	}

	//duration
	duration := service.Duration
	if duration == 0 {
		duration = DefaultResetDuration
	}

	//Construct the PasswordReser type
	pwReset := PasswordReset{
		UserID:    userID,
		Token:     token,
		TokenHash: service.hash(token),
		ExpiredAt: time.Now().Add(duration),
	}

	// Insert into the DB the pwReset
	row = service.DB.QueryRow(`
		INSERT INTO password_resets (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3) ON CONFLICT (user_id) DO
		UPDATE
		SET token_hash = $2, expires_at = $3 
		RETURNING id;`, pwReset.UserID, pwReset.TokenHash, pwReset.ExpiredAt)
	err = row.Scan(&pwReset.ID)
	if err != nil {
		return nil, fmt.Errorf("create: %w", err)
	}
	fmt.Println("---Time right now:", time.Now())
	fmt.Println("---Tokensito:", pwReset.Token)
	return &pwReset, nil
}

// We are going to consume a token and return the user associated with it, or return an error if the token wasn't valid for any reason.
func (service *PasswordResetService) Consume(token string) (*User, error) {
	tokenHash := service.hash(token)
	var user User
	var pwReset PasswordReset
	row := service.DB.QueryRow(`
	  SELECT password_resets.id, password_resets.expires_at, users.id,
	    users.email, users.name, users.date_of_birth, users.password_hash, users.username_original,
	  FROM password_resets 
	    JOIN users ON password_resets.user_id = users.id
	  WHERE password_resets.token_hash = $1`, tokenHash)
	err := row.Scan(&pwReset.ID, &pwReset.ExpiredAt, &user.ID, &user.Email, &user.Name, &user.Dob, &user.PasswordHash, &user.UsernameOriginal)
	if err != nil {
		return nil, fmt.Errorf("consume: %w", err)
	}
	if time.Now().After(pwReset.ExpiredAt) {
		return nil, fmt.Errorf("token expired: %v", token)
	}
	err = service.delete(pwReset.ID)
	if err != nil {
		return nil, fmt.Errorf("consume: %w", err)
	}
	return &user, nil
}

func (service *PasswordResetService) delete(id int) error {
	_, err := service.DB.Exec(`
		DELETE FROM password_resets
		WHERE id = $1;`, id)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

func (service *PasswordResetService) hash(token string) string {
	tokenHash := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(tokenHash[:])
}
