package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func (app *Application) routes() http.Handler {
	mux := chi.NewRouter()

	mux.Use(cors.AllowAll().Handler)

	mux.Route("/events", func(r chi.Router) {
		r.With(app.authMiddleware).Get("/{event_id}/leaderboard", app.handlers.GetEventLeaderboardEventHandler)
		r.With(app.authMiddleware).Get("/{event_id}/rooms/{room_id}/leaderboard", app.handlers.GetRoomLeaderboardEventHandler)
	})

	return mux
}
