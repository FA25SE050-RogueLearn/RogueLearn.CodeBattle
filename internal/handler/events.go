package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/events"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/pkg/request"
	"github.com/google/uuid"
)

// SSE Event Handler for room's leaderboard
// Send the room events to connected players
func (hr *HandlerRepo) GetRoomLeaderboardEventHandler(w http.ResponseWriter, r *http.Request) {
	// we will get the playerID through request
	playerID, roomID, err := getRequestPlayerIDAndRoomID(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

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
	go func() {
		roomHub.Events <- events.PlayerJoined{PlayerID: playerID, RoomID: roomID}
	}()

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
// Send the leaderboard changes of events to connected users
func (hr *HandlerRepo) GetEventLeaderboardEventHandler(w http.ResponseWriter, r *http.Request) {

}

type playerIDAndRoomIDRequest struct {
	PlayerID uuid.UUID `json:"player_id"`
	RoomID   uuid.UUID `json:"room_id"`
}

// getRequestPlayerIdAndRoomId extract player_id and room_id from query params
func getRequestPlayerIDAndRoomID(w http.ResponseWriter, r *http.Request) (playerID, roomID uuid.UUID, err error) {
	var result playerIDAndRoomIDRequest
	err = request.DecodeJSON(w, r, &result)
	if err != nil {
		return uuid.UUID{}, uuid.UUID{}, err
	}

	return result.PlayerID, result.RoomID, nil
}

type playerIDAndEventIDRequest struct {
	PlayerID uuid.UUID `json:"player_id"`
	EventID  uuid.UUID `json:"event_id"`
}

// getRequestPlayerIdAndRoomId extract player_id and room_id from query params
func getRequestPlayerIDAndEventID(w http.ResponseWriter, r *http.Request) (playerID, eventID uuid.UUID, err error) {
	var result playerIDAndEventIDRequest
	err = request.DecodeJSON(w, r, &result)
	if err != nil {
		return uuid.UUID{}, uuid.UUID{}, err
	}

	return result.PlayerID, result.EventID, nil
}
