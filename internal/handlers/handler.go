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
	"github.com/jackc/pgx/v5/pgxpool"
)

// HandlerRepo holds all the dependencies required by the handlers.
// This includes the application logger, services like the RoomManager,
// and the centralized store for data access.
type HandlerRepo struct {
	worker      *executor.WorkerPool
	eventHub    *hub.EventHub
	logger      *slog.Logger
	queries     *store.Queries
	db          *pgxpool.Pool // Add db pool for transactions
	jwtParser   *jwt.JWTParser
	codeBuilder executor.CodeBuilder
}

// NewHandlerRepo creates a new HandlerRepo with the provided dependencies.
func NewHandlerRepo(logger *slog.Logger, db *pgxpool.Pool, queries *store.Queries, codeBuilder executor.CodeBuilder, worker *executor.WorkerPool) *HandlerRepo {
	secKey := env.GetString("JWT_SECRET_KEY", "")
	if secKey == "" {
		panic("JWT_SECRET_KEY env not found")
	}
	return &HandlerRepo{
		worker:      worker,
		logger:      logger,
		db:          db,
		queries:     queries,
		jwtParser:   jwt.NewJWTParser(secKey, logger),
		eventHub:    hub.NewEventHub(queries, logger, codeBuilder, worker),
		codeBuilder: codeBuilder,
	}
}

func toPgtypeUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: id,
		Valid: true,
	}
}
