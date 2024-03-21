package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"twitter-clone/controllers"
	"twitter-clone/migrations"
	"twitter-clone/models"
	"twitter-clone/templates"
	"twitter-clone/views"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"
	"github.com/joho/godotenv"
)

type config struct {
	PSQL models.PostgresConfig
	SMTP models.SMTPConfig
	CSRF struct {
		Key    string
		Secure bool
	}
	Server struct {
		Address string
	}
}

func loadConfig() (config, error) {
	var cfg config
	err := godotenv.Load()
	if err != nil {
		return cfg, err
	}
	// TODO: Setup PSQL config
	cfg.PSQL = models.PostgresConfig{
		Host:     os.Getenv("PSQL_HOST"),
		Port:     os.Getenv("PSQL_PORT"),
		User:     os.Getenv("PSQL_USER"),
		Password: os.Getenv("PSQL_PASSWORD"),
		Database: os.Getenv("PSQL_DATABASE"),
		SSLMode:  os.Getenv("PSQL_SSLMODE"),
	}
	if cfg.PSQL.Host == "" && cfg.PSQL.Port == "" {
		return cfg, fmt.Errorf("no psql config provided")
	}
	// TODO: Setup SMTP config
	cfg.SMTP.Host = os.Getenv("SMTP_HOST")
	strPort := os.Getenv("SMTP_PORT")
	cfg.SMTP.Port, err = strconv.Atoi(strPort)
	if err != nil {
		return cfg, err
	}
	cfg.SMTP.Username = os.Getenv("SMTP_USERNAME")
	cfg.SMTP.Password = os.Getenv("SMTP_PASSWORD")
	// TODO: Setup CSRF config
	cfg.CSRF.Key = os.Getenv("CSRF_KEY")
	cfg.CSRF.Secure = os.Getenv("CSRF_SECURE") == "true"
	// Address
	cfg.Server.Address = os.Getenv("SERVER_ADDRESS")
	return cfg, nil
}

func main() {
	// Set up a database connection
	cfg, err := loadConfig()
	if err != nil {
		panic(err)
	}
	err = run(cfg)
	if err != nil {
		panic(err)
	}
}
func run(cfg config) error {
	db, err := models.Open(cfg.PSQL)
	if err != nil {
		return err
	}
	defer db.Close()

	err = models.MigrateFS(db, migrations.FS, ".")
	if err != nil {
		return err
	}

	// Set up services
	userService := &models.UserService{
		DB: db,
	}
	sessionService := &models.SessionService{
		DB: db,
	}
	passwordResetService := &models.PasswordResetService{
		DB: db,
	}
	emailService := models.NewEmailService(cfg.SMTP)
	tweetService := &models.TweetService{
		DB: db,
	}
	imageService := &models.ImageService{
		ImagesDir: "images",
	}
	bookmarkService := &models.BookmarkService{
		DB: db,
	}

	// Set up middleware
	umw := controllers.UserMiddleware{
		SessionService: sessionService,
	}

	csrfMw := csrf.Protect(
		[]byte(cfg.CSRF.Key),
		// TODO: Fix this before deploying
		csrf.Secure(cfg.CSRF.Secure),
		csrf.Path("/"),
	)

	// Set up controllers
	usersC := controllers.User{
		UserService:          userService,
		SessionService:       sessionService,
		PasswordResetService: passwordResetService,
		EmailService:         emailService,
		ImageService:         imageService,
	}
	tweetsC := controllers.Tweet{
		TweetService:    tweetService,
		UserService:     userService,
		ImageService:    imageService,
		BookmarkService: bookmarkService,
	}
	imagesC := controllers.Image{
		ImageService: imageService,
	}
	bookmarksC := controllers.Bookmark{
		BookmarkService: bookmarkService,
		TweetService:    tweetService,
	}

	// Set up controllers/templates
	usersC.Templates.New = views.Must(views.ParseFS(templates.FS, "signup.gohtml", "auth_tailwind.gohtml"))
	usersC.Templates.SignIn = views.Must(views.ParseFS(templates.FS, "signin.gohtml", "auth_tailwind.gohtml"))
	usersC.Templates.ForgotPassword = views.Must(views.ParseFS(templates.FS, "forgot-pw.gohtml", "auth_tailwind.gohtml"))
	usersC.Templates.CheckYourEmail = views.Must(views.ParseFS(templates.FS, "check-your-email.gohtml", "auth_tailwind.gohtml"))
	usersC.Templates.ResetPassword = views.Must(views.ParseFS(templates.FS, "reset-pw.gohtml", "auth_tailwind.gohtml"))
	usersC.Templates.Followers = views.Must(views.ParseFS(templates.FS, "followers.gohtml", "tailwind.gohtml"))
	usersC.Templates.Following = views.Must(views.ParseFS(templates.FS, "following.gohtml", "tailwind.gohtml"))

	tweetsC.Templates.Home = views.Must(views.ParseFS(templates.FS, "home.gohtml", "tailwind.gohtml"))
	tweetsC.Templates.Errors = views.Must(views.ParseFS(templates.FS, "errors-tweet.gohtml"))
	tweetsC.Templates.SingleTweet = views.Must(views.ParseFS(templates.FS, "single-tweet.gohtml", "tailwind.gohtml"))
	tweetsC.Templates.Profile = views.Must(views.ParseFS(templates.FS, "profile.gohtml", "tailwind.gohtml"))

	bookmarksC.Templates.Bookmark = views.Must(views.ParseFS(templates.FS, "bookmarks.gohtml", "tailwind.gohtml"))

	// Set up our router
	r := chi.NewRouter()

	// These middleware are used everywhere.
	r.Use(csrfMw)
	r.Use(umw.SetUser)
	r.Use(middleware.Logger)

	// Now we set up routes.
	r.Get("/", controllers.StaticHandler(views.Must(views.ParseFS(templates.FS, "root.gohtml"))))

	r.Get("/signup", usersC.New)
	r.Post("/users", usersC.Create)
	r.Get("/signin", usersC.SignIn)
	r.Post("/signin", usersC.ProcessSignIn)
	r.Post("/signout", usersC.ProcessSignOut)
	r.Get("/forgot-pw", usersC.ForgotPassword)
	r.Post("/forgot-pw", usersC.ProcessForgotPassword)
	r.Get("/reset-pw", usersC.ResetPassword)
	r.Post("/reset-pw", usersC.ProcessResetPassword)

	r.Get("/images/{dir}/{filename}", imagesC.RenderImages)
	// /images/tweets/{tweetID}/photo-{idx}
	r.Get("/images/tweets/{tweetID}/{filename}", imagesC.RenderTweetImages)
	// /images/replies/{tweetID}/{replyID}/photo-{idx}
	r.Get("/images/replies/{tweetID}/{replyID}/{filename}", imagesC.RenderReplyImages)

	r.With(umw.RequireUser).Get("/home", tweetsC.TweetsByUserFollowing)
	r.With(umw.RequireUser).Get("/messages", controllers.StaticHandler(views.Must(views.ParseFS(templates.FS, "messages.gohtml", "tailwind.gohtml"))))
	r.With(umw.RequireUser).Get("/bookmarks", bookmarksC.RenderBookmarks)

	r.With(umw.RequireUser).Get("/{username}", tweetsC.Profile)
	r.Post("/following", usersC.CreateFollowing)
	r.Post("/unfollowing", usersC.Unfollowing)
	r.Get("/{username}/followers", usersC.ShowFollowers)
	r.Get("/{username}/following", usersC.ShowFollowing)
	r.Post("/{username}/profile-image", usersC.EditProfile)

	r.Post("/tweets", tweetsC.Create)
	r.Post("/tweets/ajax", tweetsC.CreateAjax)
	r.Get("/{username}/status/{tweetID}", tweetsC.RenderSingleTweet)
	r.Post("/like-dislike-tweet", tweetsC.HandleLikeDislikeTweet)
	r.Get("/get-tweet-data-count", tweetsC.GetTweetDataCountHandler)
	r.Post("/retweet", tweetsC.HandleRetweet)

	r.Post("/bookmark", bookmarksC.Create)

	r.Route("/users/me", func(r chi.Router) {
		r.Use(umw.RequireUser)
		r.Get("/", usersC.CurrentUser)
	})

	assetsHandler := http.FileServer(http.Dir("assets"))
	r.Get("/assets/*", http.StripPrefix("/assets", assetsHandler).ServeHTTP)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Page not found", http.StatusNotFound)
	})
	// Start the server
	fmt.Printf("Starting the server on %s...\n", cfg.Server.Address)
	return http.ListenAndServe(cfg.Server.Address, r)
}
