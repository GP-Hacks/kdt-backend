package main

import (
	"context"
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
	log.Info("Configuration loaded")
	log.Info("Logger loaded")

	grpcServer := grpc.NewServer()
	l, err := net.Listen("tcp", cfg.LocalAddress)
	if err != nil {
		log.Error("Failed to start listener for VotesService", slog.String("error", err.Error()), slog.String("address", cfg.LocalAddress))
		return
	}
	defer l.Close()

	storage, err := storage.NewPostgresStorage(cfg.PostgresAddress + "?sslmode=disable")
	if err != nil {
		log.Error("Failed to connect to Postgres", slog.String("error", err.Error()), slog.String("storage_path", cfg.PostgresAddress))
		return
	}
	log.Info("Postgres connected")
	defer storage.Close()

	_, err = storage.DB.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS votes (
			id SERIAL PRIMARY KEY,
			category VARCHAR(255),
			name TEXT,
			description TEXT,
			organization TEXT,
			photo TEXT,
			end_time TIMESTAMP
		)
	`)
	if err != nil {
		log.Error("Failed to create table", slog.String("error", err.Error()))
	}

	_, err = storage.DB.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS options (
			vote_id INT REFERENCES votes(id) ON DELETE CASCADE,
			option VARCHAR(255)
		)
	`)
	if err != nil {
		log.Error("Failed to create table", slog.String("error", err.Error()))
	}

	_, err = storage.DB.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS rate_results (
			vote_id INT REFERENCES votes(id) ON DELETE CASCADE,
			user_token TEXT,
			rate INT,
			UNIQUE (vote_id, user_token)
		)
	`)

	if err != nil {
		log.Error("Failed to create table", slog.String("error", err.Error()))
	}

	_, err = storage.DB.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS petition_results (
			vote_id INT REFERENCES votes(id) ON DELETE CASCADE,
			user_token TEXT,
			support VARCHAR(50),
			UNIQUE (vote_id, user_token)
		)
	`)

	if err != nil {
		log.Error("Failed to create table", slog.String("error", err.Error()))
	}

	_, err = storage.DB.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS choices_results (
			vote_id INT REFERENCES votes(id) ON DELETE CASCADE,
			user_token TEXT,
			choice TEXT,
			UNIQUE (vote_id, user_token)
		)
	`)

	if err != nil {
		log.Error("Failed to create table", slog.String("error", err.Error()))
	}

	if err := storage.FetchAndStoreData(context.Background()); err != nil {
		log.Error("Failed to fetch and store data", slog.String("error", err.Error()))
		return
	}

	handler.NewGRPCHandler(cfg, grpcServer, storage, log)
	if err := grpcServer.Serve(l); err != nil {
		log.Error("Error serving gRPC server for VotesService", slog.String("address", cfg.LocalAddress), slog.String("error", err.Error()))
	}
}
