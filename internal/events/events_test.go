package events

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestEventTypes(t *testing.T) {
	t.Run("Event type constants are correctly defined", func(t *testing.T) {
		assert.Equal(t, EventType("CORRECT_SOLUTION_SUBMITTED"), CORRECT_SOLUTION_SUBMITTED)
		assert.Equal(t, EventType("WRONG_SOLUTION_SUBMITTED"), WRONG_SOLUTION_SUBMITTED)
		assert.Equal(t, EventType("SOLUTION_SUBMITTED"), SOLUTION_SUBMITTED)
		assert.Equal(t, EventType("PLAYER_JOINED"), PLAYER_JOINED)
		assert.Equal(t, EventType("PLAYER_LEFT"), PLAYER_LEFT)
		assert.Equal(t, EventType("ROOM_DELETED"), ROOM_DELETED)
		assert.Equal(t, EventType("COMPILATION_TEST"), COMPILATION_TEST)
	})

	t.Run("Event types are strings", func(t *testing.T) {
		var eventType EventType = "CUSTOM_EVENT"
		assert.IsType(t, EventType(""), eventType)
		assert.Equal(t, "CUSTOM_EVENT", string(eventType))
	})
}

func TestJudgeStatus(t *testing.T) {
	t.Run("Judge status constants are correctly defined", func(t *testing.T) {
		assert.Equal(t, JudgeStatus("Accepted"), Accepted)
		assert.Equal(t, JudgeStatus("Wrong Answer"), WrongAnswer)
		assert.Equal(t, JudgeStatus("Runtime Error"), RuntimeError)
		assert.Equal(t, JudgeStatus("Compilation Error"), CompilationError)
		assert.Equal(t, JudgeStatus("Time Limit Exceeded"), TimeLimitExceeded)
		assert.Equal(t, JudgeStatus("Memory Limit Exceeded"), MemoryLimitExceeded)
	})

	t.Run("Judge statuses are strings", func(t *testing.T) {
		var status JudgeStatus = "Custom Status"
		assert.IsType(t, JudgeStatus(""), status)
		assert.Equal(t, "Custom Status", string(status))
	})
}

func TestSseEvent(t *testing.T) {
	t.Run("Create SSE event with string data", func(t *testing.T) {
		event := SseEvent{
			EventType: PLAYER_JOINED,
			Data:      "player-123 joined",
		}

		assert.Equal(t, PLAYER_JOINED, event.EventType)
		assert.Equal(t, "player-123 joined", event.Data)
	})

	t.Run("Create SSE event with struct data", func(t *testing.T) {
		playerData := map[string]interface{}{
			"player_id": "123",
			"room_id":   "456",
		}

		event := SseEvent{
			EventType: PLAYER_JOINED,
			Data:      playerData,
		}

		assert.Equal(t, PLAYER_JOINED, event.EventType)
		assert.Equal(t, playerData, event.Data)
	})

	t.Run("Create SSE event with nil data", func(t *testing.T) {
		event := SseEvent{
			EventType: ROOM_DELETED,
			Data:      nil,
		}

		assert.Equal(t, ROOM_DELETED, event.EventType)
		assert.Nil(t, event.Data)
	})
}

func TestSolutionSubmitted(t *testing.T) {
	t.Run("Create valid SolutionSubmitted event", func(t *testing.T) {
		playerID := uuid.New()
		eventID := uuid.New()
		roomID := uuid.New()
		problemID := uuid.New()
		submittedTime := time.Now()

		event := SolutionSubmitted{
			PlayerID:      playerID,
			EventID:       eventID,
			RoomID:        roomID,
			ProblemID:     problemID,
			Code:          "func solution() { return 42; }",
			Language:      "go",
			SubmittedTime: submittedTime,
		}

		assert.Equal(t, playerID, event.PlayerID)
		assert.Equal(t, eventID, event.EventID)
		assert.Equal(t, roomID, event.RoomID)
		assert.Equal(t, problemID, event.ProblemID)
		assert.Equal(t, "func solution() { return 42; }", event.Code)
		assert.Equal(t, "go", event.Language)
		assert.Equal(t, submittedTime, event.SubmittedTime)
	})

	t.Run("Create SolutionSubmitted with empty fields", func(t *testing.T) {
		event := SolutionSubmitted{
			Code:     "",
			Language: "",
		}

		assert.Empty(t, event.Code)
		assert.Empty(t, event.Language)
		assert.Equal(t, uuid.Nil, event.PlayerID)
		assert.True(t, event.SubmittedTime.IsZero())
	})
}

func TestSolutionResult(t *testing.T) {
	t.Run("Create SolutionResult with Accepted status", func(t *testing.T) {
		playerID := uuid.New()
		eventID := uuid.New()
		roomID := uuid.New()
		problemID := uuid.New()

		solutionSubmitted := SolutionSubmitted{
			PlayerID:      playerID,
			EventID:       eventID,
			RoomID:        roomID,
			ProblemID:     problemID,
			Code:          "correct solution",
			Language:      "python",
			SubmittedTime: time.Now(),
		}

		result := SolutionResult{
			SolutionSubmitted: solutionSubmitted,
			Score:             100,
			Status:            Accepted,
			Message:           "All test cases passed",
		}

		assert.Equal(t, solutionSubmitted, result.SolutionSubmitted)
		assert.Equal(t, 100, result.Score)
		assert.Equal(t, Accepted, result.Status)
		assert.Equal(t, "All test cases passed", result.Message)
	})

	t.Run("Create SolutionResult with Wrong Answer status", func(t *testing.T) {
		playerID := uuid.New()

		solutionSubmitted := SolutionSubmitted{
			PlayerID: playerID,
			Code:     "incorrect solution",
			Language: "java",
		}

		result := SolutionResult{
			SolutionSubmitted: solutionSubmitted,
			Score:             0,
			Status:            WrongAnswer,
			Message:           "Test case 1 failed: expected 5, got 3",
		}

		assert.Equal(t, solutionSubmitted, result.SolutionSubmitted)
		assert.Equal(t, 0, result.Score)
		assert.Equal(t, WrongAnswer, result.Status)
		assert.Contains(t, result.Message, "Test case 1 failed")
	})

	t.Run("Create SolutionResult with Runtime Error status", func(t *testing.T) {
		solutionSubmitted := SolutionSubmitted{
			Code:     "buggy solution",
			Language: "cpp",
		}

		result := SolutionResult{
			SolutionSubmitted: solutionSubmitted,
			Score:             0,
			Status:            RuntimeError,
			Message:           "Segmentation fault",
		}

		assert.Equal(t, RuntimeError, result.Status)
		assert.Equal(t, "Segmentation fault", result.Message)
		assert.Equal(t, 0, result.Score)
	})

	t.Run("Create SolutionResult with negative score", func(t *testing.T) {
		result := SolutionResult{
			Score:  -10,
			Status: CompilationError,
		}

		assert.Equal(t, -10, result.Score)
		assert.Equal(t, CompilationError, result.Status)
	})
}

func TestLeaderboardUpdated(t *testing.T) {
	t.Run("Create LeaderboardUpdated event", func(t *testing.T) {
		roomID := uuid.New()

		event := LeaderboardUpdated{
			RoomID: roomID,
		}

		assert.Equal(t, roomID, event.RoomID)
	})

	t.Run("Create LeaderboardUpdated with nil UUID", func(t *testing.T) {
		event := LeaderboardUpdated{
			RoomID: uuid.Nil,
		}

		assert.Equal(t, uuid.Nil, event.RoomID)
	})
}

func TestPlayerJoined(t *testing.T) {
	t.Run("Create PlayerJoined event", func(t *testing.T) {
		playerID := uuid.New()
		roomID := uuid.New()

		event := PlayerJoined{
			PlayerID: playerID,
			RoomID:   roomID,
		}

		assert.Equal(t, playerID, event.PlayerID)
		assert.Equal(t, roomID, event.RoomID)
	})

	t.Run("Create PlayerJoined with same player and room ID", func(t *testing.T) {
		id := uuid.New()

		event := PlayerJoined{
			PlayerID: id,
			RoomID:   id,
		}

		assert.Equal(t, id, event.PlayerID)
		assert.Equal(t, id, event.RoomID)
	})
}

func TestPlayerLeft(t *testing.T) {
	t.Run("Create PlayerLeft event", func(t *testing.T) {
		playerID := uuid.New()
		roomID := uuid.New()

		event := PlayerLeft{
			PlayerID: playerID,
			RoomID:   roomID,
		}

		assert.Equal(t, playerID, event.PlayerID)
		assert.Equal(t, roomID, event.RoomID)
	})

	t.Run("PlayerLeft and PlayerJoined have same structure", func(t *testing.T) {
		playerID := uuid.New()
		roomID := uuid.New()

		joinedEvent := PlayerJoined{
			PlayerID: playerID,
			RoomID:   roomID,
		}

		leftEvent := PlayerLeft{
			PlayerID: playerID,
			RoomID:   roomID,
		}

		assert.Equal(t, joinedEvent.PlayerID, leftEvent.PlayerID)
		assert.Equal(t, joinedEvent.RoomID, leftEvent.RoomID)
	})
}

func TestRoomDeleted(t *testing.T) {
	t.Run("Create RoomDeleted event", func(t *testing.T) {
		roomID := uuid.New()

		event := RoomDeleted{
			RoomID: roomID,
		}

		assert.Equal(t, roomID, event.RoomID)
	})

	t.Run("Create RoomDeleted with zero UUID", func(t *testing.T) {
		event := RoomDeleted{
			RoomID: uuid.UUID{},
		}

		assert.Equal(t, uuid.UUID{}, event.RoomID)
		assert.True(t, event.RoomID == uuid.Nil)
	})
}

func TestEventInteractions(t *testing.T) {
	t.Run("Multiple events with same IDs", func(t *testing.T) {
		playerID := uuid.New()
		roomID := uuid.New()

		joinEvent := PlayerJoined{
			PlayerID: playerID,
			RoomID:   roomID,
		}

		leftEvent := PlayerLeft{
			PlayerID: playerID,
			RoomID:   roomID,
		}

		deleteEvent := RoomDeleted{
			RoomID: roomID,
		}

		assert.Equal(t, joinEvent.PlayerID, leftEvent.PlayerID)
		assert.Equal(t, joinEvent.RoomID, leftEvent.RoomID)
		assert.Equal(t, joinEvent.RoomID, deleteEvent.RoomID)
	})

	t.Run("Solution submission workflow", func(t *testing.T) {
		playerID := uuid.New()
		roomID := uuid.New()
		problemID := uuid.New()
		eventID := uuid.New()

		// Player joins room
		joinEvent := PlayerJoined{
			PlayerID: playerID,
			RoomID:   roomID,
		}

		// Player submits solution
		submissionEvent := SolutionSubmitted{
			PlayerID:      playerID,
			EventID:       eventID,
			RoomID:        roomID,
			ProblemID:     problemID,
			Code:          "solution code",
			Language:      "go",
			SubmittedTime: time.Now(),
		}

		// System evaluates solution
		resultEvent := SolutionResult{
			SolutionSubmitted: submissionEvent,
			Score:             85,
			Status:            Accepted,
			Message:           "Solution accepted",
		}

		// Leaderboard gets updated
		leaderboardEvent := LeaderboardUpdated{
			RoomID: roomID,
		}

		// Verify the workflow
		assert.Equal(t, joinEvent.PlayerID, submissionEvent.PlayerID)
		assert.Equal(t, joinEvent.RoomID, submissionEvent.RoomID)
		assert.Equal(t, submissionEvent, resultEvent.SolutionSubmitted)
		assert.Equal(t, submissionEvent.RoomID, leaderboardEvent.RoomID)
		assert.Equal(t, Accepted, resultEvent.Status)
		assert.Equal(t, 85, resultEvent.Score)
	})
}

// Benchmark tests
func BenchmarkSolutionSubmittedCreation(b *testing.B) {
	playerID := uuid.New()
	eventID := uuid.New()
	roomID := uuid.New()
	problemID := uuid.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SolutionSubmitted{
			PlayerID:      playerID,
			EventID:       eventID,
			RoomID:        roomID,
			ProblemID:     problemID,
			Code:          "test solution",
			Language:      "go",
			SubmittedTime: time.Now(),
		}
	}
}

func BenchmarkSseEventCreation(b *testing.B) {
	data := map[string]interface{}{
		"player_id": "123",
		"room_id":   "456",
		"timestamp": time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SseEvent{
			EventType: PLAYER_JOINED,
			Data:      data,
		}
	}
}
