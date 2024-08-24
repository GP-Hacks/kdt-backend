package main

import (
	"context"
	"github.com/GP-Hacks/kdt2024-commons/prettylogger"
	"github.com/GP-Hacks/kdt2024-votes/config"
	"github.com/GP-Hacks/kdt2024-votes/internal/grpc-server/handler"
	"github.com/GP-Hacks/kdt2024-votes/internal/storage"
	"google.golang.org/grpc"
	"log/slog"
	"net"
)

func main() {
	cfg := config.MustLoad()
	log := prettylogger.SetupLogger(cfg.Env)
	log.Info("Configuration loaded", slog.String("env", cfg.Env))
	log.Info("Logger initialized")

	grpcServer := grpc.NewServer()

	log.Info("Starting TCP listener", slog.String("address", cfg.Address))
	l, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		log.Error("Failed to start TCP listener for VotesService", slog.String("error", err.Error()), slog.String("address", cfg.Address))
		return
	}
	defer func() {
		if err := l.Close(); err != nil {
			log.Error("Failed to close TCP listener", slog.String("error", err.Error()))
		}
	}()
	log.Info("TCP listener started successfully", slog.String("address", cfg.Address))

	storage, err := setupPostgreSQL(cfg, log)

	handler.NewGRPCHandler(cfg, grpcServer, storage, log)
	if err := grpcServer.Serve(l); err != nil {
		log.Error("Error serving gRPC server for VotesService", slog.String("address", cfg.Address), slog.String("error", err.Error()))
	}
}

func setupPostgreSQL(cfg *config.Config, log *slog.Logger) (*storage.PostgresStorage, error) {
	storage, err := storage.NewPostgresStorage(cfg.PostgresAddress + "?sslmode=disable")
	if err != nil {
		log.Error("Failed to connect to PostgreSQL", slog.String("error", err.Error()), slog.String("postgres_address", cfg.PostgresAddress))
		return nil, err
	}
	log.Info("PostgreSQL connected", slog.String("postgres_address", cfg.PostgresAddress))

	if err := storage.CreateTables(context.Background()); err != nil {
		log.Error("Error creating tables", slog.String("error", err.Error()))
		return nil, err
	}
	log.Info("Tables created or already exist")

	log.Info("Fetching and storing initial data")
	if err := storage.FetchAndStoreData(context.Background()); err != nil {
		log.Error("Failed to fetch and store initial data", slog.String("error", err.Error()))
		return nil, err
	}
	log.Info("Initial data fetched and stored successfully")
	return storage, nil
}
