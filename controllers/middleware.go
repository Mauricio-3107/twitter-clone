package controllers

import (
	"log"
	"net/http"
	"twitter-clone/context"
	"twitter-clone/models"
)

type UserMiddleware struct {
	SessionService *models.SessionService
}

func (umw UserMiddleware) SetUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := readCookie(r, CookieSession)
		if err != nil {
			log.Printf("Error reading cookie: %v", err)
			next.ServeHTTP(w, r)
			return
		}
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}

		user, err := umw.SessionService.User(token)
		if err != nil {
			log.Printf("Error retrieving user from session: %v", err)
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		ctx = context.WithUser(ctx, user)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func (umw UserMiddleware) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := context.User(r.Context())

		if user == nil {
			http.Redirect(w, r, "/signin", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}
