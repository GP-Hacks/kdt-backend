package main

import (
	"context"
	"fmt"
	"github.com/GP-Hack/kdt2024-commons/prettylogger"
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

	l, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		log.Error("Failed to start TCP listener for VotesService", slog.String("error", err.Error()), slog.String("address", cfg.Address))
		return
	}
	defer closeListener(l, log)

	storage, err := storage.NewPostgresStorage(cfg.PostgresAddress+"?sslmode=disable", log)
	if err != nil {
		log.Error("Failed to connect to PostgreSQL", slog.String("error", err.Error()), slog.String("postgres_address", cfg.PostgresAddress))
		return
	}
	log.Info("PostgreSQL connected", slog.String("postgres_address", cfg.PostgresAddress))

	if err := createTables(storage, log); err != nil {
		log.Error("Error creating tables", slog.String("error", err.Error()))
		return
	}

	if err := storage.FetchAndStoreData(context.Background()); err != nil {
		log.Error("Failed to fetch and store initial data", slog.String("error", err.Error()))
		return
	}
	log.Info("Initial data fetched and stored")

	handler.NewGRPCHandler(cfg, grpcServer, storage, log)
	if err := grpcServer.Serve(l); err != nil {
		log.Error("Error serving gRPC server for VotesService", slog.String("address", cfg.Address), slog.String("error", err.Error()))
	}
}

func closeListener(l net.Listener, log *slog.Logger) {
	if err := l.Close(); err != nil {
		log.Error("Error closing TCP listener", slog.String("error", err.Error()))
	} else {
		log.Info("TCP listener closed")
	}
}

func createTables(storage *storage.PostgresStorage, log *slog.Logger) error {
	tables := []struct {
		name  string
		query string
	}{
		{
			name: "votes",
			query: `
				CREATE TABLE IF NOT EXISTS votes (
					id SERIAL PRIMARY KEY,
					category VARCHAR(255),
					name TEXT,
					description TEXT,
					organization TEXT,
					photo TEXT,
					end_time TIMESTAMP
				)`,
		},
		{
			name: "options",
			query: `
				CREATE TABLE IF NOT EXISTS options (
					vote_id INT REFERENCES votes(id) ON DELETE CASCADE,
					option VARCHAR(255)
				)`,
		},
		{
			name: "rate_results",
			query: `
				CREATE TABLE IF NOT EXISTS rate_results (
					vote_id INT REFERENCES votes(id) ON DELETE CASCADE,
					user_token TEXT,
					rate INT,
					UNIQUE (vote_id, user_token)
				)`,
		},
		{
			name: "petition_results",
			query: `
				CREATE TABLE IF NOT EXISTS petition_results (
					vote_id INT REFERENCES votes(id) ON DELETE CASCADE,
					user_token TEXT,
					support VARCHAR(50),
					UNIQUE (vote_id, user_token)
				)`,
		},
		{
			name: "choices_results",
			query: `
				CREATE TABLE IF NOT EXISTS choices_results (
					vote_id INT REFERENCES votes(id) ON DELETE CASCADE,
					user_token TEXT,
					choice TEXT,
					UNIQUE (vote_id, user_token)
				)`,
		},
	}

	for _, table := range tables {
		_, err := storage.DB.Exec(context.Background(), table.query)
		if err != nil {
			return fmt.Errorf("failed to create table %s: %w", table.name, err)
		}
		log.Info("Table created or already exists", slog.String("table", table.name))
	}
	return nil
}
