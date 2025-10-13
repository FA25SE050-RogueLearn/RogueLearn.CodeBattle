package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/events"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/store"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// SSE Event Handler for room's leaderboard
// Send the Room events to connected players
func (hr *HandlerRepo) JoinRoomHandler(w http.ResponseWriter, r *http.Request) {
	// we will get the playerID through request
	eventID, roomID, err := getRequestEventIDAndRoomID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hr.logger.Info("joined to the room",
		"event_id", eventID,
		"room_id", roomID)

	// get from the passed context in production stage, we will get from query in dev stage
	playerIDStr := r.URL.Query().Get("player_id")
	playerID, err := uuid.Parse(playerIDStr)
	if err != nil {
		hr.logger.Error("failed to parse playerID",
			"player_id", playerIDStr)
		hr.badRequest(w, r, err)
		return
	}

	hr.logger.Info("player join requested",
		"player_id", playerID)

	hr.logger.Info("rooms map", "rooms_map", hr.eventHub.Rooms)

	// Set http headers required for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Get the room manager for the requested room.
	roomHub := hr.eventHub.GetRoomById(roomID)
	if roomHub == nil {
		http.Error(w, "room not found or not active", http.StatusNotFound)
		return
	}

	// listen for incoming SseEvents
	listen := make(chan events.SseEvent) // Add buffer to prevent blocking

	// Properly lock when modifying listeners
	roomHub.Mu.Lock()
	if roomHub.Listerners == nil {
		roomHub.Listerners = make(map[uuid.UUID]chan<- events.SseEvent)
	}
	roomHub.Listerners[playerID] = listen
	roomHub.Mu.Unlock()

	defer hr.logger.Info("SSE connection closed", "player_id", playerID, "room_id", roomID)
	defer close(listen)
	defer func() {
		roomHub.Mu.Lock()
		delete(roomHub.Listerners, playerID)
		roomHub.Mu.Unlock()
		go func() {
			roomHub.Events <- events.PlayerLeft{PlayerID: playerID, RoomID: roomID}
		}()
	}()

	hr.logger.Info("SSE connection established", "player_id", playerID, "room_id", roomID)

	// player joined event
	roomHub.Events <- events.PlayerJoined{PlayerID: playerID, RoomID: roomID}

	for {
		select {
		case <-r.Context().Done():
			hr.logger.Info("SSE client disconnected", "player_id", playerID, "room_id", roomID)
			// player left event
			return
		case event, ok := <-listen:
			if !ok {
				hr.logger.Info("SSE client disconnected", "player_id", playerID, "room_id", roomID)
				return
			}

			hr.logger.Info("Sending event to player's client", "player_id", playerID, "event", event, "room_id", roomID)
			data, err := json.Marshal(event)
			if err != nil {
				hr.logger.Error("failed to marshal SSE event", "error", err, "player_id", playerID)
				return // Client is likely gone, so exit
			}

			if event.EventType != "" {
				fmt.Fprintf(w, "event: %s\n", event.EventType)
			}

			fmt.Fprintf(w, "data: %s\n\n", string(data))

			w.(http.Flusher).Flush()
		}
	}
}

// SSE Event Handler for event's leaderboards
// Send the Event's changes of events to connected users
func (hr *HandlerRepo) SpectateEventHandler(w http.ResponseWriter, r *http.Request) {
}

func (hr *HandlerRepo) GetEventsHandler(w http.ResponseWriter, r *http.Request) {
	// For now, no pagination.
	// In the future, we can add helper functions to parse query params for pagination.
	params := store.GetEventsParams{
		Limit:  10,
		Offset: 0,
	}

	events, err := hr.queries.GetEvents(r.Context(), params)
	if err != nil {
		hr.serverError(w, r, err)
		return
	}

	err = response.JSON(w, response.JSONResponseParameters{
		Status:  http.StatusOK,
		Data:    events,
		Success: true,
		Msg:     "Events retrieved successfully",
	})
	if err != nil {
		hr.serverError(w, r, err)
	}
}

func (hr *HandlerRepo) GetEventRoomsHandler(w http.ResponseWriter, r *http.Request) {
	eventIDStr := chi.URLParam(r, "event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		hr.badRequest(w, r, errors.New("invalid event ID format"))
		return
	}

	pgEventID := pgtype.UUID{Bytes: eventID, Valid: true}

	rooms, err := hr.queries.GetRoomsByEvent(r.Context(), pgEventID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			hr.notFound(w, r)
		} else {
			hr.serverError(w, r, err)
		}
		return
	}

	err = response.JSON(w, response.JSONResponseParameters{
		Status:  http.StatusOK,
		Data:    rooms,
		Success: true,
		Msg:     "Rooms for the event retrieved successfully",
	})

	if err != nil {
		hr.serverError(w, r, err)
	}
}

// getRequestPlayerIdAndRoomId extract player_id and room_id from query params
func getRequestEventIDAndRoomID(r *http.Request) (eventID, roomID uuid.UUID, err error) {
	eventIDStr := chi.URLParam(r, "event_id")
	roomIDStr := chi.URLParam(r, "room_id")
	eventIDUID, err := uuid.Parse(eventIDStr)
	if err != nil {
		return uuid.UUID{}, uuid.UUID{}, err
	}

	roomIDUID, err := uuid.Parse(roomIDStr)
	if err != nil {
		return uuid.UUID{}, uuid.UUID{}, err
	}

	return eventIDUID, roomIDUID, nil
}

// getRequestPlayerIdAndRoomId extract player_id and room_id from query params
func getRequestPlayerIDAndEventID(r *http.Request) (playerID, eventID uuid.UUID, err error) {
	playerIDStr := r.URL.Query().Get("player_id")
	eventIDStr := r.URL.Query().Get("event_id")
	playerIDUID, err := uuid.Parse(playerIDStr)
	if err != nil {
		return uuid.UUID{}, uuid.UUID{}, err
	}

	eventIDUID, err := uuid.Parse(eventIDStr)
	if err != nil {
		return uuid.UUID{}, uuid.UUID{}, err
	}

	return playerIDUID, eventIDUID, nil
}
