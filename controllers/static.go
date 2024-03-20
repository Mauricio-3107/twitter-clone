package controllers

import (
	"net/http"
	"twitter-clone/context"
)

func StaticHandler(tpl Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tpl.Execute(w, r, nil)
	}
}

func BookmarksHandler(tpl Template) http.HandlerFunc {
	bookmarks := []struct {
		Username string
		Date     string
		Text     string
		Image    string
	}{
		{
			Username: "user-1",
			Date:     "January 19, 2024",
			Text:     "Unpopular opinion: Elon is doing his job as the CEO of Tesla and the company is executing well",
			Image:    "",
		},
		{
			Username: "user-2",
			Date:     "December 23, 2022",
			Text:     "“how lazy are you?” me:",
			Image:    "/images/bookmarks/meme.jpg",
		},
		{
			Username: "user-3",
			Date:     "July 5, 2024",
			Text:     "About to drive this 👀",
			Image:    "/images/bookmarks/meme.jpg",
		},
	}
	_ = bookmarks
	return func(w http.ResponseWriter, r *http.Request) {
		usernameCtx := context.User(r.Context()).UsernameOriginal
		var data struct {
			Username string
			TweetIDs []int
		}
		data.Username = usernameCtx
		tpl.Execute(w, r, data)
	}
}
