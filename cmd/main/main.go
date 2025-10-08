package main

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"

	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/cmd/api"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/database"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/protos"
	"google.golang.org/grpc"

	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/service"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/store"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/pkg/env"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/pkg/jwt"
	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	// dburl := env.GetString("DATABASE_URL", "")
	// if dburl == "" {
	// 	log.Fatal("DATABASE_URL not found")
	// }

	cfg := &api.Config{Port: 8080}

	// test area
	connStr := env.GetString("SUPABASE_DB_CONNECTION_STRING", "")
	if connStr == "" {
		panic("SUPABASE_DB_CONNECTION_STRING environment variable is not set")
	}

	db, err := database.NewPool(connStr)
	if err != nil {
		panic(err)
	}

	queries := store.New(db)

	// log to os standard output
	slogHandler := tint.NewHandler(os.Stdout, &tint.Options{Level: slog.LevelDebug, AddSource: true})
	logger := slog.New(slogHandler)
	slog.SetDefault(logger) // Set default for any library using slog's default logger

	// worker, err := executor.NewWorkerPool(logger, queries, &executor.WorkerPoolOptions{
	// 	MaxWorkers:       5,
	// 	MemoryLimitBytes: 256,
	// 	MaxJobCount:      3,
	// 	CpuNanoLimit:     5000,
	// })
	// if err != nil {
	// 	panic(err)
	// }
	// gr := channels.NewGlobalRooms(queries, logger, worker)

	// handlerRepo := handlers.NewHandlerRepo(logger, gr, queries)

	// app := &Application{
	// 	cfg:     cfg,
	// 	logger:  logger,
	// 	queries: queries,
	// }

	token := "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJPbmxpbmUgSldUIEJ1aWxkZXIiLCJpYXQiOjE3NTk4NTYwMjEsImV4cCI6MTc5MTM5MjAyMSwiYXVkIjoid3d3LmV4YW1wbGUuY29tIiwic3ViIjoianJvY2tldEBleGFtcGxlLmNvbSIsImlkIjoiSm9obm55IiwidXNlcm5hbWUiOiJqb25oeXNpbnMiLCJlbWFpbCI6Impyb2NrZXRAZXhhbXBsZS5jb20iLCJyb2xlIjpbIk1hbmFnZXIiLCJQcm9qZWN0IEFkbWluaXN0cmF0b3IiXX0.WaZwRkpgCxIYSqEfv8SD2Q3TlXsJRJiaMStlbJfDDos"
	secKey := "qwertyuiopasdfghjklzxcvbnm123456"
	jwtParser := jwt.NewJWTParser(secKey, logger)
	claims, err := jwtParser.GetUserClaimsFromToken(token)
	if err != nil {
		logger.Error("Failed to parse jwt, exiting application...", "err", err)
		panic(err)
	}

	logger.Info("Parsed successfully!", "claims", claims)

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", cfg.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	protos.RegisterCodeBattleServiceServer(grpcServer, service.NewCodeBattleServer(queries, logger))

	grpcServer.Serve(lis)

	// err = app.run()
	// if err != nil {
	// 	// Using standard log here to be absolutely sure it prints if slog itself had an issue
	// 	log.Printf("CRITICAL ERROR from run(): %v\n", err)
	// 	currentTrace := string(debug.Stack())
	// 	log.Printf("Trace: %s\n", currentTrace)
	// 	// Also log with slog if it's available
	// 	slog.Error("CRITICAL ERROR from run()", "error", err.Error(), "trace", currentTrace)
	// 	os.Exit(1)
	// }
}
