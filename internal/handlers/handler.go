package handlers

import (
	"log/slog"

	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/executor"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/hub"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/store"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/pkg/env"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/pkg/jwt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// HandlerRepo holds all the dependencies required by the handlers.
// This includes the application logger, services like the RoomManager,
// and the centralized store for data access.
type HandlerRepo struct {
	eventHub  *hub.EventHub
	logger    *slog.Logger
	queries   *store.Queries
	jwtParser *jwt.JWTParser
}

// NewHandlerRepo creates a new HandlerRepo with the provided dependencies.
func NewHandlerRepo(logger *slog.Logger, queries *store.Queries) *HandlerRepo {
	secKey := env.GetString("JWT_SECRET_KEY", "")
	if secKey == "" {
		panic("JWT_SECRET_KEY env not found")
	}
	return &HandlerRepo{
		logger:    logger,
		queries:   queries,
		jwtParser: jwt.NewJWTParser(secKey, logger),
		eventHub:  hub.NewEventHub(queries, logger, &executor.WorkerPool{}),
	}
}

func toPgtypeUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: id,
		Valid: true,
	}
}
