package handlers

import (
	"log/slog"

	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/hub"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/store"
)

// HandlerRepo holds all the dependencies required by the handlers.
// This includes the application logger, services like the RoomManager,
// and the centralized store for data access.
type HandlerRepo struct {
	eventHub *hub.EventHub
	logger   *slog.Logger
	queries  *store.Queries
}

// NewHandlerRepo creates a new HandlerRepo with the provided dependencies.
func NewHandlerRepo(logger *slog.Logger, queries *store.Queries) *HandlerRepo {
	return &HandlerRepo{
		logger:  logger,
		queries: queries,
	}
}
