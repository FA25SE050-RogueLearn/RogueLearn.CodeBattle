package handlers

import (
	"database/sql"
	"net/http"

	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (hr *HandlerRepo) GetRoomProblemsHandler(w http.ResponseWriter, r *http.Request) {
	roomIDStr := chi.URLParam(r, "event_id")
	roomIDUID, err := uuid.Parse(roomIDStr)
	if err != nil {
		hr.badRequest(w, r, err)
		return
	}

	cps, err := hr.queries.GetEventCodeProblems(r.Context(), toPgtypeUUID(roomIDUID))
	if err != nil {
		if err == sql.ErrNoRows {
			hr.logger.Info("event code problems not found")
			hr.notFound(w, r)
			return
		}
		hr.serverError(w, r, err)
		return
	}

	hr.logger.Info("event code problems found", "event_code_problems", cps)

	err = response.JSON(w, response.JSONResponseParameters{
		Status:  http.StatusOK,
		Success: true,
		Msg:     "get code problems successfully",
		Data:    cps,
	})
	if err != nil {
		hr.logger.Error("failed to parse json", "err", err)
		hr.serverError(w, r, err)
	}
}
