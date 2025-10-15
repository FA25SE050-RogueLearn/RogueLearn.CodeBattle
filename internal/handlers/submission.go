package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/events"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/executor"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/store"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/pkg/request"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	DefaultQueryTimeoutSecond = 10 * time.Second
)

var (
	ErrLanguageNotFound error = errors.New("Invalid programming language")
	ErrInvalidProblem   error = errors.New("Invalid problem")
	ErrWrongSyntax      error = errors.New("Wrong syntax")
)

type SubmissionRequest struct {
	ProblemID string `json:"problem_id"`
	Code      string `json:"code"`
	Language  string `json:"language"`
}

type SubmissionResponse struct {
	Stdout        string           `json:"stdout"`
	Stderr        string           `json:"stderr"`
	Message       string           `json:"message"`
	Success       bool             `json:"success"`
	Error         executor.CodeErr `json:"error"`
	ExecutionTime string           `json:"execution_time_ms"`
}

// SubmitSolutionHandler will compile and run test cases of a solution for a code problem
func (hr *HandlerRepo) SubmitSolutionHandler(w http.ResponseWriter, r *http.Request) {
	var req SubmissionRequest
	err := request.DecodeJSON(w, r, &req)
	if err != nil {
		hr.badRequest(w, r, ErrInvalidRequest)
		return
	}

	// get player_id through query param in development stage
	playerIDStr := r.URL.Query().Get("player_id")
	playerID, err := uuid.Parse(playerIDStr)
	if err != nil {
		hr.badRequest(w, r, errors.New("failed to parse player_id"))
		return
	}

	problemIDUID, err := uuid.Parse(req.ProblemID)
	if err != nil {
		hr.badRequest(w, r, ErrInvalidProblem)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultQueryTimeoutSecond)
	defer cancel()

	normalizedLang, found := executor.NormalizeLanguage(req.Language)
	if !found {
		hr.logger.Warn("programming language not found", "lang", req.Language)
		hr.badRequest(w, r, ErrLanguageNotFound)
		return
	}

	lang, err := hr.queries.GetLanguageByName(ctx, normalizedLang)
	if err != nil {
		hr.logger.Error("failed to get language", "lang", normalizedLang)
		return
	}

	problem, err := hr.queries.GetCodeProblemLanguageDetail(ctx, store.GetCodeProblemLanguageDetailParams{
		CodeProblemID: toPgtypeUUID(problemIDUID),
		LanguageID:    lang.ID,
	})
	if err != nil {
		hr.logger.Error("Error", "question", err)
		hr.badRequest(w, r, ErrInternalServer)
		return
	}

	testCases, err := hr.queries.GetTestCasesByProblem(ctx, problem.CodeProblemID)
	if err != nil {
		hr.logger.Error("failed to get test cases", "problem_id", problem.CodeProblemID)
		hr.badRequest(w, r, ErrInternalServer)
		return
	}

	// combine problem's driver code with user's code
	finalCode, err := hr.codeBuilder.Build(normalizedLang, problem.DriverCode, req.Code)
	if err == executor.ErrParsed {
		hr.logger.Warn("Wrong syntax")
		hr.badRequest(w, r, ErrWrongSyntax)
		return
	}

	if err != nil {
		hr.logger.Error("failed to build code", "err", err)
		hr.serverError(w, r, ErrInternalServer)
		return
	}

	hr.logger.Info("Code built successfully", "final_code", finalCode)

	s, err := hr.queries.CreateSubmission(r.Context(), store.CreateSubmissionParams{
		UserID:        toPgtypeUUID(playerID),
		LanguageID:    lang.ID,
		CodeProblemID: toPgtypeUUID(problemIDUID),
		CodeSubmitted: req.Code,
		Status:        store.SubmissionStatusPending,
	})

	result := hr.worker.ExecuteJob(lang, finalCode, testCases)

	err = hr.updateSubmissionStatus(ctx, s.ID, result)
	if err != nil {
		hr.logger.Error("failed to update submission status", "err", err)
		hr.serverError(w, r, ErrInternalServer)
		return
	}

	response.JSON(w, response.JSONResponseParameters{
		Status: http.StatusOK,
		Data: SubmissionResponse{
			Stdout:        result.Stdout,
			Stderr:        result.Stderr,
			Message:       result.Message,
			Success:       result.Success,
			Error:         result.Error,
			ExecutionTime: result.ExecutionTime,
		},
		Success: true,
		Msg:     "solution submitted successfully.",
		ErrMsg:  "",
	})
}

func (hr *HandlerRepo) updateSubmissionStatus(ctx context.Context, submissionID pgtype.UUID, result executor.Result) error {
	var status store.SubmissionStatus

	if result.Success {
		status = store.SubmissionStatusAccepted
	} else {
		// 2. Use a switch on the error type for clarity.
		switch result.Error {
		case executor.RunTimeError:
			status = store.SubmissionStatusRuntimeError
		case executor.CompileError:
			status = store.SubmissionStatusCompilationError
		case executor.FailTestCase:
			status = store.SubmissionStatusWrongAnswer
		default:
			// 3. Add a default case to catch nil or unknown errors.
			// This prevents the zero-value bug.
			status = store.SubmissionStatusRuntimeError // A safe default for unknown failures.
		}
	}

	_, err := hr.queries.UpdateSubmissionStatus(ctx, store.UpdateSubmissionStatusParams{
		ID:     submissionID,
		Status: status,
	})
	if err != nil {
		return err
	}

	return nil
}

func (hr *HandlerRepo) SubmitSolutionInRoomHandler(w http.ResponseWriter, r *http.Request) {
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

	var reqPayload SubmissionRequest
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
