package handlers

import (
	"database/sql"
	"net/http"

	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/executor"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/store"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type CodeProblemResponse struct {
	Title            string `json:"title"`
	ProblemStatement string `json:"problem_statement"`
	Difficulty       int32  `json:"difficulty"`
}

func (hr *HandlerRepo) GetProblemsHandler(w http.ResponseWriter, r *http.Request) {
	// For now, no pagination.
	// In the future, we can add helper functions to parse query params for pagination.
	params := store.GetCodeProblemsParams{
		Limit:  10,
		Offset: 0,
	}

	problems, err := hr.queries.GetCodeProblems(r.Context(), params)
	if err != nil {
		hr.serverError(w, r, err)
		return
	}

	err = response.JSON(w, response.JSONResponseParameters{
		Status:  http.StatusOK,
		Data:    problems,
		Success: true,
		Msg:     "Problems retrieved successfully",
	})
	if err != nil {
		hr.serverError(w, r, err)
	}
}

func (hr *HandlerRepo) GetProblemHandler(w http.ResponseWriter, r *http.Request) {
	pIDStr := chi.URLParam(r, "problem_id")
	pIDUID, err := uuid.Parse(pIDStr)
	if err != nil {
		hr.badRequest(w, r, ErrInvalidRequest)
		return
	}

	problem, err := hr.queries.GetCodeProblemByID(r.Context(), toPgtypeUUID(pIDUID))
	if err != nil {
		hr.serverError(w, r, err)
		return
	}

	err = response.JSON(w, response.JSONResponseParameters{
		Status:  http.StatusOK,
		Data:    toProblemResponse(problem),
		Success: true,
		Msg:     "Problems retrieved successfully",
	})
	if err != nil {
		hr.serverError(w, r, err)
	}
}

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

type CodeProblemLanguageDetailResponse struct {
	SolutionStub      string `json:"solution_stub"`
	DriverCode        string `json:"driver_code"`
	TimeConstraintMs  int32  `json:"time_constraint_ms"`
	SpaceConstraintMb int32  `json:"space_constraint_mb"`
}

func (hr *HandlerRepo) GetProblemDetails(w http.ResponseWriter, r *http.Request) {
	problemIDStr := chi.URLParam(r, "problem_id")
	problemID, err := uuid.Parse(problemIDStr)
	if err != nil {
		hr.badRequest(w, r, err)
		return
	}

	normalizedLang, found := executor.NormalizeLanguage(r.URL.Query().Get("lang"))
	if !found {
		hr.logger.Warn("lang not found")
		hr.notFound(w, r)
		return
	}

	detail, err := hr.queries.GetCodeProblemLanguageDetailByLanguageName(r.Context(), store.GetCodeProblemLanguageDetailByLanguageNameParams{
		CodeProblemID: toPgtypeUUID(problemID),
		Name:          normalizedLang,
	})

	if err != nil {
		hr.logger.Error("failed to get code problem language detail", "err", err)
		hr.serverError(w, r, err)
		return
	}

	err = response.JSON(w, response.JSONResponseParameters{
		Status:  http.StatusOK,
		Success: true,
		Msg:     "get code problem language detail successfully",
		Data:    toProblemDetailResponse(detail),
	})
	if err != nil {
		hr.logger.Error("failed to parse json", "err", err)
		hr.serverError(w, r, err)
	}
}

func toProblemResponse(problem store.CodeProblem) CodeProblemResponse {
	return CodeProblemResponse{
		Title:            problem.Title,
		ProblemStatement: problem.ProblemStatement,
		Difficulty:       problem.Difficulty,
	}
}

func toProblemDetailResponse(problem store.CodeProblemLanguageDetail) CodeProblemLanguageDetailResponse {
	return CodeProblemLanguageDetailResponse{
		SolutionStub:      problem.SolutionStub,
		DriverCode:        problem.DriverCode,
		TimeConstraintMs:  problem.TimeConstraintMs,
		SpaceConstraintMb: problem.SpaceConstraintMb,
	}
}
