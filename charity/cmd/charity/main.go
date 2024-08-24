package main

import (
	"context"
	"github.com/GP-Hacks/kdt2024-charity/config"
	"github.com/GP-Hacks/kdt2024-charity/internal/grpc-server/handler"
	"github.com/GP-Hacks/kdt2024-charity/internal/storage"
	"github.com/GP-Hacks/kdt2024-commons/prettylogger"
	"github.com/streadway/amqp"
	"google.golang.org/grpc"
	"log/slog"
	"net"
)

func main() {
	cfg := config.MustLoad()
	log := prettylogger.SetupLogger(cfg.Env)
	log.Info("Configuration and logger initialized", slog.String("environment", cfg.Env))
	log.Info("Logger initialized")

	grpcServer := grpc.NewServer()

	log.Info("Starting TCP listener", slog.String("address", cfg.Address))
	l, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		log.Error("Failed to start TCP listener for CharityService", slog.String("error", err.Error()), slog.String("address", cfg.Address))
		return
	}
	defer func() {
		if err := l.Close(); err != nil {
			log.Error("Failed to close TCP listener", slog.String("error", err.Error()))
		}
	}()
	log.Info("TCP listener started successfully", slog.String("address", cfg.Address))

	storage, err := setupPostgreSQL(cfg, log)
	if err != nil {
		return
	}

	conn, ch, err := setupRabbitMQ(cfg, log)
	if err != nil {
		return
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Error("Failed to close RabbitMQ connection", slog.String("error", err.Error()))
		}
		if err := ch.Close(); err != nil {
			log.Error("Failed to close RabbitMQ channel", slog.String("error", err.Error()))
		}
	}()

	handler.NewGRPCHandler(cfg, grpcServer, storage, log, ch)
	serveGRPC(grpcServer, l, log, cfg)
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

func setupRabbitMQ(cfg *config.Config, log *slog.Logger) (*amqp.Connection, *amqp.Channel, error) {
	log.Info("Connecting to RabbitMQ", slog.String("address", cfg.RabbitMQAddress))
	conn, err := amqp.Dial(cfg.RabbitMQAddress)
	if err != nil {
		log.Error("Failed to connect to RabbitMQ", slog.String("error", err.Error()), slog.String("address", cfg.RabbitMQAddress))
		return nil, nil, err
	}
	log.Info("RabbitMQ connection established successfully", slog.String("address", cfg.RabbitMQAddress))

	ch, err := conn.Channel()
	if err != nil {
		log.Error("Failed to open RabbitMQ channel", slog.String("error", err.Error()))
		return nil, nil, err
	}
	log.Info("RabbitMQ channel opened successfully")
	return conn, ch, nil
}

func serveGRPC(grpcServer *grpc.Server, l net.Listener, log *slog.Logger, cfg *config.Config) {
	log.Info("Starting gRPC server", slog.String("address", cfg.Address))
	if err := grpcServer.Serve(l); err != nil {
		log.Error("Error serving gRPC server", slog.String("error", err.Error()), slog.String("address", cfg.Address))
	} else {
		log.Info("gRPC server started successfully", slog.String("address", cfg.Address))
	}
}
