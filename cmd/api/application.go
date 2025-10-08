package api

import (
	"log/slog"
	"sync"

	handlers "github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/handler"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/store"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/pkg/jwt"
)

type Application struct {
	wg        sync.WaitGroup
	cfg       *Config
	handlers  *handlers.HandlerRepo
	logger    *slog.Logger
	queries   *store.Queries
	jwtParser *jwt.JWTParser
}

type Config struct {
	Port int
}
