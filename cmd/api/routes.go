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
		// Public routes for events
		r.Get("/", app.handlers.GetEventsHandler)
		r.Get("/{event_id}/rooms", app.handlers.GetEventRoomsHandler)

		// Auth-protected routes for event interaction
		r.Get("/{event_id}/leaderboard", app.handlers.SpectateEventHandler)
		r.Get("/{event_id}/rooms/{room_id}/leaderboard", app.handlers.JoinRoomHandler)
		r.Post("/{event_id}/rooms/{room_id}/submit", app.handlers.SubmitSolutionHandler)
		r.Get("/{event_id}/rooms/{room_id}/problems", app.handlers.GetRoomProblemsHandler)
	})

	return mux
}
