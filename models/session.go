package models

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"twitter-clone/rand"
)

const (
	// The minimum number of bytes to be used for each session token.
	MinBytesPerToken = 32
)

type Session struct {
	ID     int
	UserID int
	// Token is inly set when creating a new session. When we look up a session, this will be left empty, as we only store the hash of a session token in our DB and we cannot reverse it into a raw token
	Token     string
	TokenHash string
}

type SessionService struct {
	DB *sql.DB
	// BytesPerToken is used to determine how many bytes to use when generating each session token. If this value is not set or is less than the MinBytesPerToken const it will be ignored and MinBytesPerToken will be used.
	BytesPerToken int
}

func (ss *SessionService) Create(userID int) (*Session, error) {
	bytesPerToken := ss.BytesPerToken
	if bytesPerToken < MinBytesPerToken {
		bytesPerToken = MinBytesPerToken
	}
	token, err := rand.String(bytesPerToken)
	if err != nil {
		fmt.Println(err)
		return nil, fmt.Errorf("create: %w", err)
	}
	session := Session{
		Token:     token,
		UserID:    userID,
		TokenHash: ss.hash(token),
	}
	row := ss.DB.QueryRow(`
		INSERT INTO
    		sessions (user_id, token_hash)
		VALUES
    		($1, $2) ON CONFLICT (user_id) DO
		UPDATE
		SET
    		token_hash = $2 RETURNING id;`, session.UserID, session.TokenHash)
	err = row.Scan(&session.ID)
	if err != nil {
		return nil, fmt.Errorf("create: %w", err)
	}
	return &session, nil
}

func (ss *SessionService) User(token string) (*User, error) {
	var user User
	// 1. Hash the token
	tokenHash := ss.hash(token)
	// 2. Query the session for that hash
	row := ss.DB.QueryRow(`
		SELECT users.id, users.email, users.name, users.date_of_birth, users.password_hash, users.username_original, users.profile_image_url
		FROM sessions
		JOIN users ON users.id = sessions.user_id
		WHERE sessions.token_hash = $1;`, tokenHash)
	err := row.Scan(&user.ID, &user.Email, &user.Name, &user.Dob, &user.PasswordHash, &user.UsernameOriginal, &user.ProfileImage)
	if err != nil {
		return nil, fmt.Errorf("user: %w", err)
	}

	// 4. Return the user
	return &user, nil
}

func (ss *SessionService) Delete(token string) error {
	tokenHash := ss.hash(token)
	_, err := ss.DB.Exec(`
	  DELETE FROM sessions
	  WHERE token_hash = $1;`, tokenHash)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

func (ss *SessionService) hash(token string) string {
	tokenHash := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(tokenHash[:])
}
