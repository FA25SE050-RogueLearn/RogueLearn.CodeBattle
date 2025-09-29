package main

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"sync"

	pb "github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/api"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/database"
	"google.golang.org/grpc"

	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/service"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/internal/store"
	"github.com/FA25SE050-RogueLearn/RogueLearn.CodeBattle/pkg/env"
	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
)

type Application struct {
	wg      sync.WaitGroup
	cfg     *Config
	logger  *slog.Logger
	queries *store.Queries
}

type Config struct {
	Port int
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	// dburl := env.GetString("DATABASE_URL", "")
	// if dburl == "" {
	// 	log.Fatal("DATABASE_URL not found")
	// }

	cfg := &Config{Port: 8080}

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

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", cfg.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterCodeBattleServiceServer(grpcServer, service.NewCodeBattleServer(queries, logger))
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
