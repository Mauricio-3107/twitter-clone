package controllers

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"
	"twitter-clone/context"
	"twitter-clone/errors"
	"twitter-clone/models"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"
)

type User struct {
	Templates struct {
		New            Template
		SignIn         Template
		ForgotPassword Template
		CheckYourEmail Template
		ResetPassword  Template
		Followers      Template
		Following      Template
	}
	UserService          *models.UserService
	SessionService       *models.SessionService
	PasswordResetService *models.PasswordResetService
	EmailService         *models.EmailService
	ImageService         *models.ImageService
}

func (u User) New(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email     string
		CSRFField template.HTML
	}
	data.Email = r.FormValue("email")
	data.CSRFField = csrf.TemplateField(r)

	u.Templates.New.Execute(w, r, data)
}

func (u User) Create(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")
	birthday := r.FormValue("birthday")
	username := r.FormValue("username")

	// Input validation
	if name == "" || email == "" || password == "" || birthday == "" || username == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	layout := "2006-01-02"
	createdDate, err := time.Parse(layout, birthday)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong with the birthday", http.StatusInternalServerError)
		return
	}
	// Set the time part to midnight (00:00:00)
	createdDate = createdDate.UTC().Truncate(24 * time.Hour)

	nu := models.NewUser{
		Name:     name,
		Email:    email,
		Password: password,
		Dob:      createdDate,
		Username: username,
	}

	user, err := u.UserService.Create(nu)
	if err != nil {
		if errors.Is(err, models.ErrEmailTaken) {
			err = errors.Public(err, "Esa cuenta de email ya está asociada a otra cuenta")
		} else if errors.Is(err, models.ErrUsernameTaken) {
			err = errors.Public(err, "Ese nombre de usuario ya está asociado a otra cuenta")
		}
		u.Templates.New.Execute(w, r, nu, err)
		return
	}

	session, err := u.SessionService.Create(user.ID)
	if err != nil {
		fmt.Println(err)
		// TODO: Long term, we should show a warning about not being able to sign the user in.
		http.Redirect(w, r, "/signin", http.StatusFound)
		return
	}
	setCookie(w, CookieSession, session.Token)
	http.Redirect(w, r, "/home", http.StatusFound)
}

func (u User) SignIn(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email string
	}
	data.Email = r.FormValue("email")
	u.Templates.SignIn.Execute(w, r, data)
}

func (u User) ProcessSignIn(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	// Input validation
	if email == "" || password == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	user, err := u.UserService.Authenticate(email, password)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Wrong email or password", http.StatusBadRequest)
		return
	}
	session, err := u.SessionService.Create(user.ID)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}

	setCookie(w, CookieSession, session.Token)
	http.Redirect(w, r, "/home", http.StatusFound)
}

func (u User) CurrentUser(w http.ResponseWriter, r *http.Request) {
	// SetUser and RequireUser middleware are required.
	user := context.User(r.Context())
	fmt.Fprintf(w, "Current user: %+v\n", user.Email)
}

func (u User) ProcessSignOut(w http.ResponseWriter, r *http.Request) {
	token, err := readCookie(r, CookieSession)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusFound)
		return
	}
	err = u.SessionService.Delete(token)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	deleteCookie(w, CookieSession)

	http.Redirect(w, r, "/signin", http.StatusFound)
}

func (u User) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email string
	}
	data.Email = r.FormValue("email")

	u.Templates.ForgotPassword.Execute(w, r, data)
}

func (u User) ProcessForgotPassword(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email string
	}
	email := r.FormValue("email")
	if email == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}
	data.Email = email
	pwdReset, err := u.PasswordResetService.Create(data.Email)
	if err != nil {
		// TODO: Handle other cases in the future. For instance,
		// if a user doesn't exist with the email address.
		fmt.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}

	vals := url.Values{
		"token": {pwdReset.Token},
	}
	resetURL := "http://localhost:3000/reset-pw?" + vals.Encode()
	err = u.EmailService.ForgotPassword(data.Email, resetURL)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}
	// Don't render the token here! We need them to confirm they have access to their email to get the token. Sharing it here would be a massive security hole.
	u.Templates.CheckYourEmail.Execute(w, r, data)
}

func (u User) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Token string
	}
	data.Token = r.FormValue("token")

	u.Templates.ResetPassword.Execute(w, r, data)
}

func (u User) ProcessResetPassword(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Token    string
		Password string
	}
	data.Token = r.FormValue("token")
	data.Password = r.FormValue("password")

	user, err := u.PasswordResetService.Consume(data.Token)
	if err != nil {
		fmt.Println(err)
		// TODO: Distinguish between server errors and invalid token errors.
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}

	// Update the user's password.
	err = u.UserService.UpdatePassword(user.ID, data.Password)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}

	// Sign the user in now that they have reset their password.
	// Any errors from this point onward should redirect to the sign in page.
	session, err := u.SessionService.Create(user.ID)
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/signin", http.StatusFound)
		return
	}
	setCookie(w, CookieSession, session.Token)
	http.Redirect(w, r, "/home", http.StatusFound)
}

func (u User) EditProfile(w http.ResponseWriter, r *http.Request) {
	// Retrieve the username param
	username := chi.URLParam(r, "username")
	// Retrieve the context user
	usernameCtx := context.User(r.Context()).UsernameOriginal
	// Retrieve the context name user
	nameCtx := context.User(r.Context()).Name
	nameForm := r.FormValue("edit-name")
	// retrieve user ID
	userID := context.User(r.Context()).ID

	if username != usernameCtx {
		http.Error(w, "No estas permitido realizar esta acción", http.StatusForbidden)
		return
	}

	err := r.ParseMultipartForm(5 << 20) // 5mb
	if err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	// No file and no edited name
	if len(r.MultipartForm.File) == 0 && nameCtx == nameForm {
		// No files uploaded, handle accordingly
		editPath := fmt.Sprintf("/%s", usernameCtx)
		http.Redirect(w, r, editPath, http.StatusFound)
		return
	}

	if len(r.MultipartForm.File) != 0 && nameCtx == nameForm {
		// Yes file and no edited name
		file, fileHeader, err := r.FormFile("profile-image")
		if err != nil {
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
		defer file.Close()
		// Print debug info
		err = u.ImageService.CreateProfileImage(usernameCtx, fileHeader.Filename, file)
		if err != nil {
			var fileErr models.FileError
			if errors.As(err, &fileErr) {
				msg := fmt.Sprintf("%v has an invalid content type or extension. Only png, gif, and jpg files can be uploaded.", fileHeader.Filename)
				http.Error(w, msg, http.StatusBadRequest)
				return
			}
			http.Error(w, "Something went wrong creating the image", http.StatusInternalServerError)
			return
		}

		// Updating the profile image
		formattedFilename := fmt.Sprintf("profile-img-%s.jpeg", username)
		profileImage, err := u.ImageService.Image("users", formattedFilename)
		if err != nil {
			http.Error(w, "Something went wrong updating the profile image", http.StatusInternalServerError)
			return
		}
		profileImageURL := formatSlashes(profileImage.Path)

		err = u.UserService.UpdateProfileImage(userID, profileImageURL)
		if err != nil {
			http.Error(w, "Something went wrong parsing the form edit profile", http.StatusInternalServerError)
			return
		}
	} else if len(r.MultipartForm.File) == 0 && nameCtx != nameForm {
		// No file and yes edited name
		err = u.UserService.UpdateProfileName(userID, nameForm)
		if err != nil {
			http.Error(w, "Something went wrong parsing the form edit profile", http.StatusInternalServerError)
			return
		}
	} else {
		// Yes file and yes edited name
		file, fileHeader, err := r.FormFile("profile-image")
		if err != nil {
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
		defer file.Close()
		// Print debug info
		err = u.ImageService.CreateProfileImage(usernameCtx, fileHeader.Filename, file)
		if err != nil {
			var fileErr models.FileError
			if errors.As(err, &fileErr) {
				msg := fmt.Sprintf("%v has an invalid content type or extension. Only png, gif, and jpg files can be uploaded.", fileHeader.Filename)
				http.Error(w, msg, http.StatusBadRequest)
				return
			}
			http.Error(w, "Something went wrong creating the image", http.StatusInternalServerError)
			return
		}

		// Updating the profile image
		formattedFilename := fmt.Sprintf("profile-img-%s.jpeg", username)
		profileImage, err := u.ImageService.Image("users", formattedFilename)
		if err != nil {
			http.Error(w, "Something went wrong updating the profile image", http.StatusInternalServerError)
			return
		}
		profileImageURL := formatSlashes(profileImage.Path)

		// Update the profile image in the DB
		err = u.UserService.UpdateProfileComplete(userID, nameForm, profileImageURL)
		if err != nil {
			http.Error(w, "Something went wrong parsing the form edit profile", http.StatusInternalServerError)
			return
		}
	}
	// Since you have modified the user profile, create another session for him.
	session, err := u.SessionService.Create(userID)
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/signin", http.StatusFound)
		return
	}
	setCookie(w, CookieSession, session.Token)
	editPath := fmt.Sprintf("/%s", usernameCtx)
	http.Redirect(w, r, editPath, http.StatusFound)
}

func (u User) CreateFollowing(w http.ResponseWriter, r *http.Request) {
	userUsername := context.User(r.Context()).UsernameOriginal
	followingUsername := r.FormValue("username")

	_, err := u.UserService.CreateFollowing(userUsername, followingUsername)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	followerUsername := userUsername
	followedUsername := followingUsername
	_, err = u.UserService.CreateFollower(followedUsername, followerUsername)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	redirectURL := r.Header.Get("Referer")
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (u User) Unfollowing(w http.ResponseWriter, r *http.Request) {
	userUsername := context.User(r.Context()).UsernameOriginal
	followingUsername := r.FormValue("username")

	err := u.UserService.BreakFollowing(userUsername, followingUsername)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	followerUsername := userUsername
	followedUsername := followingUsername
	err = u.UserService.BreakFollower(followedUsername, followerUsername)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	redirectURL := r.Header.Get("Referer")
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (u User) ShowFollowers(w http.ResponseWriter, r *http.Request) {
	// Retrieve the username param
	username := chi.URLParam(r, "username")
	followers, err := u.UserService.GetFollowers(username)
	if err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	user, err := u.UserService.GetUser(username)
	if err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	type profileData struct {
		Username     string
		Name         string
		ProfileImage string
		IsFollowing  bool
		IsOwnProfile bool
	}
	type userData struct {
		Username string
		Name     string
	}
	var data struct {
		FollowersData []profileData
		UserData      userData
		TweetIDs      []int
	}
	for _, follower := range followers {
		isOwnProfile := false
		isFollowing := false

		// Retrieve the context user
		usernameCtx := context.User(r.Context()).UsernameOriginal
		if usernameCtx == follower.UsernameOriginal {
			isOwnProfile = true
		}
		if !isOwnProfile {
			// You have to read if there is a following and pass it. In the gohtml you can insert this info inside the else isOwnProfile
			checkFollowing, err := u.UserService.CheckFollowing(usernameCtx, follower.UsernameOriginal)
			if err != nil {
				fmt.Println(err)
				http.Error(w, "Something went wrong isFollowing", http.StatusInternalServerError)
				return
			}
			isFollowing = checkFollowing
		}
		data.FollowersData = append(data.FollowersData, profileData{
			Username:     follower.UsernameOriginal,
			Name:         follower.Name,
			ProfileImage: follower.ProfileImage,
			IsFollowing:  isFollowing,
			IsOwnProfile: isOwnProfile,
		})
	}
	data.UserData.Name = user.Name
	data.UserData.Username = user.UsernameOriginal

	u.Templates.Followers.Execute(w, r, data)
}

func (u User) ShowFollowing(w http.ResponseWriter, r *http.Request) {
	// Retrieve the username param
	username := chi.URLParam(r, "username")
	followings, err := u.UserService.GetFollowing(username)
	if err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	user, err := u.UserService.GetUser(username)
	if err != nil {
		http.Error(w, "Something went wrong here boy", http.StatusInternalServerError)
		return
	}
	type profileData struct {
		Username     string
		Name         string
		ProfileImage string
		IsFollowing  bool
		IsOwnProfile bool
	}
	type userData struct {
		Username string
		Name     string
	}
	var data struct {
		FollowingData []profileData
		UserData      userData
		TweetIDs      []int
	}
	for _, following := range followings {
		isOwnProfile := false
		isFollowing := false

		// Retrieve the context user
		usernameCtx := context.User(r.Context()).UsernameOriginal
		if usernameCtx == following.UsernameOriginal {
			isOwnProfile = true
		}
		if !isOwnProfile {
			// You have to read if there is a following and pass it. In the gohtml you can insert this info inside the else isOwnProfile
			checkFollowing, err := u.UserService.CheckFollowing(usernameCtx, following.UsernameOriginal)
			if err != nil {
				fmt.Println(err)
				http.Error(w, "Something went wrong isFollowing", http.StatusInternalServerError)
				return
			}
			isFollowing = checkFollowing
		}
		data.FollowingData = append(data.FollowingData, profileData{
			Username:     following.UsernameOriginal,
			Name:         following.Name,
			ProfileImage: following.ProfileImage,
			IsFollowing:  isFollowing,
			IsOwnProfile: isOwnProfile,
		})
	}
	data.UserData.Name = user.Name
	data.UserData.Username = user.UsernameOriginal

	u.Templates.Following.Execute(w, r, data)
}

func formatSlashes(input string) string {
	return strings.ReplaceAll(input, "\\", "/")
}
