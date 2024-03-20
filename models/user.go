package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID               int
	Name             string
	UsernameOriginal string
	Dob              time.Time
	Email            string
	PasswordHash     string
	ProfileImage     string
}

type NewUser struct {
	Name     string
	Username string
	Dob      time.Time
	Email    string
	Password string
}

type UserService struct {
	DB               *sql.DB
	ImageUserService ImageService
}

func (us *UserService) Create(nw NewUser) (*User, error) {
	// Input validation
	if nw.Name == "" || nw.Email == "" || nw.Password == "" || nw.Username == "" {
		return nil, fmt.Errorf("create user: all fields are required")
	}

	// Username
	LowerUsername := strings.ToLower(nw.Username)
	originalUsername := nw.Username

	// Email
	isValid := validateNewUser(nw.Email)
	if isValid != nil {
		return nil, fmt.Errorf("create user: %w", isValid)
	}
	nw.Email = strings.ToLower(nw.Email)

	// Password
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(nw.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	passwordHash := string(hashedBytes)

	// Dob
	strTime := nw.Dob.Format("2006-01-02")
	dob, err := time.Parse("2006-01-02", strTime)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// ProfileImage
	profileImage, err := us.ImageUserService.Image("assets", "user-profile-default.png")
	if err != nil {
		return nil, fmt.Errorf("create user profile image: %w", err)
	}
	profileImageURL := formatSlashes(profileImage.Path)

	user := User{
		Dob:              dob,
		Name:             nw.Name,
		Email:            nw.Email,
		PasswordHash:     passwordHash,
		UsernameOriginal: nw.Username,
		ProfileImage:     profileImageURL,
	}

	row := us.DB.QueryRow(`
	  INSERT INTO users (name, username_original, username_lower, date_of_birth, email, password_hash, profile_image_url)
	  VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id;`, user.Name, originalUsername, LowerUsername, user.Dob, user.Email, user.PasswordHash, user.ProfileImage)

	err = row.Scan(&user.ID)
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) {
			if pgError.Code == pgerrcode.UniqueViolation {
				if strings.Contains(pgError.Message, "username") {
					return nil, ErrUsernameTaken
				} else if strings.Contains(pgError.Message, "email") {
					return nil, ErrEmailTaken
				}
			}
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return &user, nil
}

func (us *UserService) Authenticate(email, password string) (*User, error) {
	// Input validation
	if email == "" || password == "" {
		return nil, fmt.Errorf("authenticate: all fields are required")
	}

	// Email
	isValid := validateNewUser(email)
	if isValid != nil {
		return nil, fmt.Errorf("authenticate: %w", isValid)
	}
	email = strings.ToLower(email)

	// Build the user
	user := User{
		Email: email,
	}
	// Retrieve the user
	row := us.DB.QueryRow(`
	  SELECT id, name, date_of_birth, password_hash, username_original
	  FROM users WHERE email = $1`, email)
	err := row.Scan(&user.ID, &user.Name, &user.Dob, &user.PasswordHash, &user.UsernameOriginal)
	if err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	// Password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	return &user, nil
}

func (us *UserService) UpdatePassword(userID int, password string) error {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	passwordHash := string(hashedBytes)
	_, err = us.DB.Exec(`
	  UPDATE users
	  SET password_hash = $2
	  WHERE id = $1;`, userID, passwordHash)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (us *UserService) UpdateProfileComplete(userID int, name, profileImage string) error {
	_, err := us.DB.Exec(`
	  UPDATE users
	  SET name = $2, profile_image_url = $3
	  WHERE id = $1;`, userID, name, profileImage)
	if err != nil {
		return fmt.Errorf("update profile user: %w", err)
	}
	return nil
}

func (us *UserService) UpdateProfileImage(userID int, profileImage string) error {
	_, err := us.DB.Exec(`
	  UPDATE users
	  SET profile_image_url = $2
	  WHERE id = $1;`, userID, profileImage)
	if err != nil {
		return fmt.Errorf("update profile user: %w", err)
	}
	return nil
}

func (us *UserService) UpdateProfileName(userID int, name string) error {
	_, err := us.DB.Exec(`
	  UPDATE users
	  SET name = $2
	  WHERE id = $1;`, userID, name)
	if err != nil {
		return fmt.Errorf("update profile user: %w", err)
	}
	return nil
}

func (us *UserService) CreateFollowing(userUsername, followingUsername string) (int, error) {
	// Normalize userUsername and followingUsername (lowecase)
	userUsername = strings.ToLower(userUsername)
	followingUsername = strings.ToLower(followingUsername)

	var idRelation int
	row := us.DB.QueryRow("INSERT INTO following (user_username, following_username) VALUES ($1, $2) RETURNING id;", userUsername, followingUsername)
	err := row.Scan(&idRelation)
	if err != nil {
		return 0, fmt.Errorf("following: %w", err)
	}
	return idRelation, nil
}

func (us *UserService) CreateFollower(userUsername, followerUsername string) (int, error) {
	// Normalize userUsername and followerUsername (lowecase)
	userUsername = strings.ToLower(userUsername)
	followerUsername = strings.ToLower(followerUsername)

	var idRelation int
	row := us.DB.QueryRow("INSERT INTO followers (user_username, follower_username) VALUES ($1, $2) RETURNING id;", userUsername, followerUsername)
	err := row.Scan(&idRelation)
	if err != nil {
		return 0, fmt.Errorf("following: %w", err)
	}
	return idRelation, nil
}

func (us *UserService) BreakFollowing(userUsername, followingUsername string) error {
	// Normalize userUsername and followingUsername (lowecase)
	userUsername = strings.ToLower(userUsername)
	followingUsername = strings.ToLower(followingUsername)

	_, err := us.DB.Exec(`
		DELETE FROM following
		WHERE user_username = $1 AND following_username = $2;
	`, userUsername, followingUsername)
	if err != nil {
		return fmt.Errorf("break following: %w", err)
	}
	return nil
}

func (us *UserService) BreakFollower(userUsername, followerUsername string) error {
	// Normalize userUsername and followerUsername (lowecase)
	userUsername = strings.ToLower(userUsername)
	followerUsername = strings.ToLower(followerUsername)

	_, err := us.DB.Exec(`
		DELETE FROM followers
		WHERE user_username = $1 AND follower_username = $2;
	`, userUsername, followerUsername)
	if err != nil {
		return fmt.Errorf("break following: %w", err)
	}
	return nil
}

func (us *UserService) CheckFollowing(userUsername, followingUsername string) (bool, error) {
	// Normalize userUsername and followingUsername (lowecase)
	userUsername = strings.ToLower(userUsername)
	followingUsername = strings.ToLower(followingUsername)

	var isFollowing bool
	row := us.DB.QueryRow(`
	  SELECT EXISTS (SELECT 1 FROM following WHERE user_username = $1 AND following_username = $2);`, userUsername, followingUsername)
	err := row.Scan(&isFollowing)
	if err != nil {
		return isFollowing, fmt.Errorf("check following: %w", err)
	}
	return isFollowing, nil
}

func (us *UserService) GetFollowingCount(userUsername string) (int, error) {
	// Normalize userUsername and followingUsername (lowecase)
	userUsername = strings.ToLower(userUsername)

	var followingCount int
	row := us.DB.QueryRow(`
        SELECT COUNT(*)
        FROM following
        WHERE user_username = $1;
    `, userUsername)
	err := row.Scan(&followingCount)
	if err != nil {
		return 0, fmt.Errorf("get following count: %w", err)
	}
	return followingCount, nil
}

func (us *UserService) GetFollowerCount(userUsername string) (int, error) {
	// Normalize userUsername and followingUsername (lowecase)
	userUsername = strings.ToLower(userUsername)

	var followerCount int
	row := us.DB.QueryRow(`
        SELECT COUNT(*)
        FROM followers
        WHERE user_username = $1;
    `, userUsername)
	err := row.Scan(&followerCount)
	if err != nil {
		return 0, fmt.Errorf("get follower count: %w", err)
	}

	return followerCount, nil
}

func (us *UserService) GetFollowersUsernames(userUsername string) ([]string, error) {
	var usernames []string
	rows, err := us.DB.Query(`
		SELECT follower_username FROM followers WHERE user_username = $1;`, userUsername)
	if err != nil {
		return nil, fmt.Errorf("get followwers usernames: %w", err)
	}
	for rows.Next() {
		var username string
		err = rows.Scan(&username)
		if err != nil {
			return nil, fmt.Errorf("get followwers usernames: %w", err)
		}
		usernames = append(usernames, username)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("get followwers usernames: %w", err)
	}
	return usernames, nil
}

func (us *UserService) GetFollowers(userUsername string) ([]User, error) {
	// Normalize userUsername and followingUsername (lowecase)
	userUsername = strings.ToLower(userUsername)

	var followers []User
	// Retrieve the username followers of the username
	umFollowers, err := us.GetFollowersUsernames(userUsername)
	if err != nil {
		return nil, fmt.Errorf("get followers: %w", err)
	}

	for _, umFollower := range umFollowers {
		var user User
		row := us.DB.QueryRow(`
	  		SELECT id, name, username_original, profile_image_url
	  		FROM users WHERE username_lower = $1`, umFollower)
		err := row.Scan(&user.ID, &user.Name, &user.UsernameOriginal, &user.ProfileImage)
		if err != nil {
			return nil, fmt.Errorf("get followers: %w", err)
		}
		followers = append(followers, user)
	}
	return followers, nil
}

func (us *UserService) GetFollowingUsernames(userUsername string) ([]string, error) {
	var usernames []string
	rows, err := us.DB.Query(`
		SELECT following_username FROM following WHERE user_username = $1;`, userUsername)
	if err != nil {
		return nil, fmt.Errorf("get following usernames: %w", err)
	}
	for rows.Next() {
		var username string
		err = rows.Scan(&username)
		if err != nil {
			return nil, fmt.Errorf("get following usernames: %w", err)
		}
		usernames = append(usernames, username)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("get followwers usernames: %w", err)
	}
	return usernames, nil
}

func (us *UserService) GetFollowing(userUsername string) ([]User, error) {
	// Normalize userUsername and followingUsername (lowecase)
	userUsername = strings.ToLower(userUsername)

	var followings []User
	// Retrieve the username followers of the username
	umFollowings, err := us.GetFollowingUsernames(userUsername)
	if err != nil {
		return nil, fmt.Errorf("get following: %w", err)
	}

	for _, umFollowing := range umFollowings {
		var user User
		row := us.DB.QueryRow(`
	  		SELECT id, name, username_original, profile_image_url
	  		FROM users WHERE username_lower = $1`, umFollowing)
		err := row.Scan(&user.ID, &user.Name, &user.UsernameOriginal, &user.ProfileImage)
		if err != nil {
			return nil, fmt.Errorf("get followers: %w", err)
		}
		followings = append(followings, user)
	}
	return followings, nil
}

func (us *UserService) GetUser(userUsername string) (User, error) {
	// Normalize userUsername and followingUsername (lowecase)
	userUsername = strings.ToLower(userUsername)

	var user User
	row := us.DB.QueryRow(`
	  	SELECT id, name, username_original, profile_image_url
	  	FROM users WHERE username_lower = $1`, userUsername)
	err := row.Scan(&user.ID, &user.Name, &user.UsernameOriginal, &user.ProfileImage)
	if err != nil {
		return User{}, fmt.Errorf("get user: %w", err)
	}

	return user, nil
}

// Helpers
func validateNewUser(email string) error {
	// Using govalidator for email validation
	if !govalidator.IsEmail(email) {
		return fmt.Errorf("invalid email address")
	}
	return nil
}

func formatSlashes(input string) string {
	return strings.ReplaceAll(input, "\\", "/")
}
