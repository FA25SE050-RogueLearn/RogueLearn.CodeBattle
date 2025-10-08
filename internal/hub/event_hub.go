package hub

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/events"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/executor"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	DefaultQueryTimeoutSecond = 10 * time.Second
)

// event-based
// each room will have a room manager, acting as a broadcaster for room-related events to all connected clients
// events is a single queue that received events from multiple sources and process it, then send to all listeners
// listeners are all the clients connected to the room, represented by their client IDs

// EventHub struct holds all the RoomHub (channel) of each room
type EventHub struct {
	worker  *executor.WorkerPool
	logger  *slog.Logger
	queries *store.Queries
	// roomID -> roomManager
	Rooms         map[uuid.UUID]*RoomHub
	Mu            sync.RWMutex
	leaderboardMu sync.Mutex // Protects leaderboard calculation
}

type RoomHub struct {
	RoomID        uuid.UUID
	Events        chan any                             // Events channel is what happened in the room
	Listerners    map[uuid.UUID]chan<- events.SseEvent // Players connected to this RoomHub
	codeBuilder   executor.CodeBuilder
	worker        *executor.WorkerPool
	logger        *slog.Logger
	queries       *store.Queries
	Mu            sync.RWMutex // Protects Listerners map
	leaderboardMu sync.Mutex   // Protects leaderboard calculation
}

func NewEventHub(queries *store.Queries, logger *slog.Logger, worker *executor.WorkerPool) *EventHub {
	e := EventHub{
		worker:  worker,
		logger:  logger,
		queries: queries,
		Rooms:   make(map[uuid.UUID]*RoomHub),
	}
	// --------- remove on production ---------
	beginnerArea, err := uuid.Parse("4d5e6f7a-8b9c-0d1e-2f3a-4b5c6d7e8f90")
	if err != nil {
		panic(err)
	}
	advancedLobby, err := uuid.Parse("5e6f7a8b-9c0d-1e2f-3a4b-5c6d7e8f90a1")
	if err != nil {
		panic(err)
	}

	// remove on production
	e.CreateRoom(beginnerArea, queries)
	e.CreateRoom(advancedLobby, queries)

	for _, r := range e.Rooms {
		go r.Start()
	}
	// --------- remove on production ---------

	return &e
}

func (h *EventHub) GetRoomById(roomID uuid.UUID) *RoomHub {
	h.Mu.RLock()
	defer h.Mu.RUnlock()
	return h.Rooms[roomID]
}

func (e *EventHub) CreateRoom(roomID uuid.UUID, queries *store.Queries) *RoomHub {
	r := newRoomHub(roomID, queries, e.worker)
	e.Mu.Lock()
	e.Rooms[roomID] = r
	e.Mu.Unlock()
	go r.Start() // Start RoomHub
	return r
}

func newRoomHub(roomId uuid.UUID, queries *store.Queries, worker *executor.WorkerPool) *RoomHub {
	pkgAnalyzer := executor.NewGoPackageAnalyzer()
	return &RoomHub{
		RoomID:        roomId,
		Events:        make(chan any, 10),
		Listerners:    make(map[uuid.UUID]chan<- events.SseEvent),
		codeBuilder:   executor.NewCodeBuilder(pkgAnalyzer),
		logger:        slog.Default(),
		queries:       queries,
		Mu:            sync.RWMutex{},
		leaderboardMu: sync.Mutex{}, // Initialize the new mutex
		worker:        worker,
	}
}

// Start will start to listen and serve events to players connected to the room
func (r *RoomHub) Start() {
	for event := range r.Events {
		switch e := event.(type) {
		case events.SolutionSubmitted:
			if err := r.processSolutionSubmitted(e); err != nil {
				r.logger.Error("failed to process solution submitted event", "error", err)
			}

		case events.SolutionResult:
			if err := r.processSolutionResult(e); err != nil {
				r.logger.Error("failed to process correct solution result event", "error", err)
			}

		case events.PlayerJoined:
			if err := r.processPlayerJoined(e); err != nil {
				r.logger.Error("failed to process player joined event", "error", err)
			}
		case events.PlayerLeft:
			if err := r.processPlayerLeft(e); err != nil {
				r.logger.Error("failed to process player left event", "error", err)
			}
		case events.RoomDeleted:
			if err := r.processRoomDeleted(e); err != nil {
				r.logger.Error("failed to process room deleted event", "error", err)
			}
		}
	}
}

func (r *RoomHub) dispatchEvent(e events.SseEvent) {
	// Safely copy listeners to avoid race conditions
	r.Mu.RLock()
	if r.Listerners == nil {
		r.Mu.RUnlock()
		r.logger.Warn("no listeners map found")
		return
	}

	listeners := make(map[uuid.UUID]chan<- events.SseEvent)
	for pid, listener := range r.Listerners {
		listeners[pid] = listener
	}
	r.Mu.RUnlock()

	r.logger.Info("Hit dispatchEvent()",
		"Number of Listeners", len(listeners),
		"Event", e)

	for playerId, listener := range listeners {
		// Capture the listener variable properly
		go func(l chan<- events.SseEvent, pid uuid.UUID) {
			defer func() {
				if a := recover(); a != nil {
					r.logger.Error("panic while dispatching event", "error", r, "player_id", pid, "event", e)
				}
			}()

			r.logger.Info("dispatching to", "player_id", pid)
			select {
			case l <- e:
				// Successfully sent
				r.logger.Info("event sent to", "player_id", pid)
			default:
				// Channel is full or closed, log but don't block
				r.logger.Warn("failed to send event to listener - channel full or closed", "player_id", pid)
			}
		}(listener, playerId)
	}
}

func (r *RoomHub) dispatchEventToPlayer(e events.SseEvent, playerID uuid.UUID) {
	r.Mu.RLock()
	if r.Listerners == nil {
		r.Mu.RUnlock()
		r.logger.Warn("no listeners map found")
		return
	}

	// find the target listener
	var listener chan<- events.SseEvent
	for pid, l := range r.Listerners {
		if pid == playerID {
			listener = l
		}
	}

	if listener == nil {
		r.logger.Error("listener not found", "player_id", playerID)
		r.Mu.RUnlock()
		return
	}

	r.Mu.RUnlock()

	// Capture the listener variable properly
	go func(l chan<- events.SseEvent, pid uuid.UUID) {
		defer func() {
			if a := recover(); a != nil {
				r.logger.Error("panic while dispatching event", "error", r, "player_id", pid, "event", e)
			}
		}()

		r.logger.Info("dispatching to", "player_id", pid)
		select {
		case l <- e:
			// Successfully sent
			r.logger.Info("event sent to", "player_id", pid)
		default:
			// Channel is full or closed, log but don't block
			r.logger.Warn("failed to send event to listener - channel full or closed", "player_id", pid)
		}
	}(listener, playerID)
}

// TODO: Rewrite processSolutionSubmitted and processSolutionResult
func (r *RoomHub) processSolutionSubmitted(event events.SolutionSubmitted) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultQueryTimeoutSecond)
	defer cancel()

	normalizedLang := executor.NormalizeLanguage(event.Language)

	lang, err := r.queries.GetLanguageByName(ctx, normalizedLang)
	if err != nil {
		r.logger.Error("error", "lang", event.Language)
		return err
	}

	problem, err := r.queries.GetCodeProblemLanguageDetail(ctx, store.GetCodeProblemLanguageDetailParams{
		CodeProblemID: toPgtypeUUID(event.ProblemID),
		LanguageID:    lang.ID,
	})
	if err != nil {
		r.logger.Error("Error", "question", err)
		return err
	}

	testCases, err := r.queries.GetTestCasesByProblem(ctx, problem.CodeProblemID)
	if err != nil {
		return err
	}

	// combine problem's driver code with user's code
	finalCode, err := r.codeBuilder.Build(normalizedLang, problem.DriverCode, event.Code)
	if err != nil {
		return err
	}

	r.logger.Info("Code built successfully", "final_code", finalCode)

	// TODO: Execute Job with input (test cases)
	// i for test cases number
	for i, tc := range testCases {
		r.logger.Info("Testing...", "test_case", tc)
		result := r.worker.ExecuteJob(lang, finalCode, &tc.Input)
		if result.Error != nil {
			r.Events <- events.SolutionResult{
				SolutionSubmitted: event,
				Status:            events.RuntimeError,
				Message:           result.Output,
			}

			// This error is the user solution's fault, so we don't return it
			return nil
		}

		// TODO: Compare output
		actualOutput := strings.TrimSpace(result.Output)
		expectedOutput := strings.TrimSpace(tc.ExpectedOutput)
		if actualOutput != expectedOutput {
			message := fmt.Sprintf("Input:%v, Expected Output:%v, Actual Output: %v", tc.Input, tc.ExpectedOutput, result.Output)
			r.logger.Warn("Output not match", "message", message)
			r.Events <- events.SolutionResult{
				SolutionSubmitted: event,
				Status:            events.WrongAnswer,
				Message:           message,
			}

			// This error is the user solution's fault, so we don't return it
			return nil
		}
		r.logger.Info("placeholder:%v", "i", i)
	}

	// eventCP, err := r.queries.GetEventCodeProblem(ctx, store.GetEventCodeProblemParams{})
	// r.Events <- events.SolutionResult{
	// 	SolutionSubmitted: event,
	// 	// Score: ,
	// 	Status:  events.Accepted,
	// 	Message: "Solution accepted",
	// }

	return nil
}

// combineCodeWithTemplate combined the userCode and templateFunction at placeHolder
func combineCodeWithTemplate(templateCode, userCode, placeHolder string) string {
	finalCode := strings.Replace(templateCode, placeHolder, userCode, 1)
	return finalCode
}

func (r *RoomHub) processSolutionResult(event events.SolutionResult) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	r.logger.Info("processSoltuionResult() hit", "event", event)

	if event.Status != events.Accepted {
		r.logger.Info("solution failed", "event", event)
		sseEvent := events.SseEvent{
			EventType: events.WRONG_SOLUTION_SUBMITTED,
			Data:      fmt.Sprintf("status:%v,message:%v", event.Status, event.Message),
		}

		// send compilation error to the player
		go r.dispatchEventToPlayer(sseEvent, event.SolutionSubmitted.PlayerID)

		return nil
	}

	r.queries.UpdateRoomPlayerScore(ctx, store.UpdateRoomPlayerScoreParams{
		RoomID: toPgtypeUUID(event.SolutionSubmitted.RoomID),
		UserID: toPgtypeUUID(event.SolutionSubmitted.PlayerID),
		// Score: ,
	})

	// Recalculate leaderboard after score update
	if err := r.calculateLeaderboard(ctx); err != nil {
		r.logger.Error("failed to calculate leaderboard after solution result", "error", err)
		// non-fatal, but should be monitored
	}

	sseEvent := events.SseEvent{
		EventType: events.CORRECT_SOLUTION_SUBMITTED,
		Data:      "",
	}

	// send event to the whole room
	go r.dispatchEvent(sseEvent)

	return nil
}

// Helper method to check if player is in room
func (r *RoomHub) playerInRoom(ctx context.Context, roomID, playerID uuid.UUID) bool {
	player, err := r.queries.GetRoomPlayer(ctx, store.GetRoomPlayerParams{
		RoomID: toPgtypeUUID(roomID),
		UserID: toPgtypeUUID(playerID),
	})

	if err == sql.ErrNoRows {
		r.logger.Error("no player found", "err", sql.ErrNoRows)
		return false
	}

	r.logger.Info("player found", "player", player)

	return true
}

// Helper method to add player to room
func (r *RoomHub) addPlayerToRoom(ctx context.Context, roomID, playerID uuid.UUID) error {
	createParams := store.CreateRoomPlayerParams{
		RoomID: toPgtypeUUID(roomID),
		UserID: toPgtypeUUID(playerID),
	}

	_, err := r.queries.CreateRoomPlayer(ctx, createParams)
	return err
}

// Helper method to remove player from room
func (r *RoomHub) removePlayerFromRoom(ctx context.Context, roomID, playerID uuid.UUID) error {
	return r.queries.DeleteRoomPlayer(ctx, store.DeleteRoomPlayerParams{
		RoomID: toPgtypeUUID(roomID),
		UserID: toPgtypeUUID(playerID),
	})
}

// calculateLeaderboard recalculates and updates player ranks in a single, atomic, and concurrency-safe operation.
func (r *RoomHub) calculateLeaderboard(ctx context.Context) error {
	// Lock to prevent concurrent calculations for the same room, which could cause deadlocks or race conditions.
	r.leaderboardMu.Lock()
	defer r.leaderboardMu.Unlock()

	r.logger.Info("Starting leaderboard calculation for room", "room_id", r.RoomID)

	// Use the new, highly efficient single query to update all ranks.
	// This avoids transactions in Go code and looping, pushing the logic to the database where it's most performant.
	err := r.queries.CalculateRoomLeaderboard(ctx, toPgtypeUUID(r.RoomID))
	if err != nil {
		r.logger.Error("Failed to update player ranks via single query", "room_id", r.RoomID, "error", err)
		return err
	}

	// r.logger.Info("Finished calculating leaderboard for room", "room_id", r.RoomID)
	return nil
}

func (r *RoomHub) processPlayerJoined(event events.PlayerJoined) error {
	// Process the player joined event
	ctx := context.Background()
	// playerID is passed by the event
	// playerID is parsed from the jwt token
	if ok := r.playerInRoom(ctx, event.RoomID, event.PlayerID); !ok {
		r.logger.Info("player is not in room, adding to room...",
			"playerID", event.PlayerID,
			"room", event.RoomID)
		err := r.addPlayerToRoom(ctx, event.RoomID, event.PlayerID)
		if err != nil {
			r.logger.Error("failed to add player to room", "error", err)
			return err
		}
	}

	// Recalculate leaderboard after a player joins
	err := r.calculateLeaderboard(ctx)
	if err != nil {
		r.logger.Error("failed to calculate leaderboard after player joined", "error", err)
		// This is not fatal to the join operation, but should be monitored.
	}

	r.logger.Info("player joined", "event", event)

	data := fmt.Sprintf("playerId:%d,roomId:%d\n\n", event.PlayerID, r.RoomID)

	sseEvent := events.SseEvent{
		EventType: events.PLAYER_JOINED,
		Data:      data,
	}

	go r.dispatchEvent(sseEvent)

	return nil
}

func (r *RoomHub) processPlayerLeft(event events.PlayerLeft) error {
	ctx := context.Background()

	// Process the player left event
	data := fmt.Sprintf("playerId:%d,roomId:%d\n\n", event.PlayerID, r.RoomID)

	err := r.removePlayerFromRoom(ctx, event.RoomID, event.PlayerID)
	if err != nil {
		r.logger.Error("failed to remove player from room", "error", err)
	}

	// Recalculate leaderboard after a player leaves
	err = r.calculateLeaderboard(ctx)
	if err != nil {
		r.logger.Error("failed to calculate leaderboard after player left", "error", err)
	}

	sseEvent := events.SseEvent{
		EventType: events.PLAYER_LEFT,
		Data:      data,
	}

	go r.dispatchEvent(sseEvent)
	r.logger.Info("player left", "event", event)

	return nil
}

// TODO: Complete this
func (r *RoomHub) processRoomDeleted(event events.RoomDeleted) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	ctx := context.Background()

	// Process the room deleted event
	data := fmt.Sprintf("roomId:%d\n\n", r.RoomID)

	sseEvent := events.SseEvent{
		EventType: events.ROOM_DELETED,
		Data:      data,
	}

	err := r.queries.DeleteRoom(ctx, toPgtypeUUID(event.RoomID))
	if err != nil {
		r.logger.Error("failed to delete room from database", "error", err)
	}
	r.logger.Info("room deleted", "roomID", event.RoomID)

	go r.dispatchEvent(sseEvent)

	return nil
}

func toPgtypeUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: id,
		Valid: true,
	}
}
