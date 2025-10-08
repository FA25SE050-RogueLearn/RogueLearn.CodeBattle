package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/events"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/pkg/request"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type submissionRequest struct {
	ProblemID string `json:"problem_id"`
	Code      string `json:"code"`
	Language  string `json:"language"`
}

func (hr *HandlerRepo) SubmitSolutionHandler(w http.ResponseWriter, r *http.Request) {
	// claims, ok := r.Context().Value("asd").(*jwt.UserClaims)
	// if !ok {
	// 	// hr.unauthorizedResponse(w, r)
	// 	return
	// }

	// get player_id through query param on dev stage.
	playerIDStr := r.URL.Query().Get("player_id")
	playerID, err := uuid.Parse(playerIDStr)
	if err != nil {
		hr.badRequest(w, r, errors.New("invalid user ID in token"))
		return
	}

	eventIDStr := chi.URLParam(r, "event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		hr.badRequest(w, r, errors.New("invalid event ID in URL"))
		return
	}

	roomIDStr := chi.URLParam(r, "room_id")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		hr.badRequest(w, r, errors.New("invalid room ID in URL"))
		return
	}

	var reqPayload submissionRequest
	err = request.DecodeJSON(w, r, &reqPayload)
	if err != nil {
		hr.badRequest(w, r, err)
		return
	}

	problemID, err := uuid.Parse(reqPayload.ProblemID)
	if err != nil {
		hr.badRequest(w, r, errors.New("invalid problem ID in request body"))
		return
	}

	roomHub := hr.eventHub.GetRoomById(roomID)
	if roomHub == nil {
		hr.notFound(w, r)
		return
	}

	submissionEvent := events.SolutionSubmitted{
		PlayerID:      playerID,
		EventID:       eventID,
		RoomID:        roomID,
		ProblemID:     problemID,
		Code:          reqPayload.Code,
		Language:      reqPayload.Language,
		SubmittedTime: time.Now(),
	}

	// Use a select with a default case to avoid blocking if the channel is full
	select {
	case roomHub.Events <- submissionEvent:
		// Event sent successfully
	default:
		hr.logger.Warn("event hub channel is full, submission dropped", "room_id", roomID)
		hr.errorMessage(w, r, http.StatusServiceUnavailable, "Server is busy, please try again later.", nil)
		return
	}

	err = response.JSON(w, response.JSONResponseParameters{
		Status:  http.StatusAccepted,
		Success: true,
		Msg:     "Solution submitted successfully and is being processed.",
	})
	if err != nil {
		hr.serverError(w, r, err)
	}
}
