package main

import (
	"context"
	"github.com/GP-Hack/kdt2024-commons/prettylogger"
	"github.com/GP-Hack/kdt2024-places/config"
	"github.com/GP-Hack/kdt2024-places/internal/grpc-server/handler"
	"github.com/GP-Hack/kdt2024-places/internal/storage"
	"github.com/streadway/amqp"
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
		log.Error("Failed to start listener for PlacesService", slog.String("error", err.Error()), slog.String("address", cfg.LocalAddress))
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
		CREATE TABLE IF NOT EXISTS places (
			id SERIAL PRIMARY KEY,
			category VARCHAR(255),
			description TEXT,
			latitude DOUBLE PRECISION,
			longitude DOUBLE PRECISION,
			location TEXT,
			name VARCHAR(255),
			tel VARCHAR(50),
			website VARCHAR(255),
			cost INT,
			time VARCHAR(50)
		)
	`)
	if err != nil {
		log.Error("Failed to create table", slog.String("error", err.Error()))
	}

	_, err = storage.DB.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS photos (
			place_id INT REFERENCES places(id) ON DELETE CASCADE,
			url TEXT
		)
	`)
	if err != nil {
		log.Error("Failed to create table", slog.String("error", err.Error()))
	}

	if err := storage.FetchAndStoreData(context.Background()); err != nil {
		log.Error("Failed to fetch and store data", slog.String("error", err.Error()))
		return
	}

	conn, err := amqp.Dial(cfg.RabbitMQAddress)
	if err != nil {
		log.Error("Failed to connect to RabbitMQ", slog.Any("error", err.Error()))
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Error("Failed to open a channel", slog.Any("error", err.Error()))
		return
	}
	defer ch.Close()

	queueName := cfg.QueuePurchases
	_, err = ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Error("Failed to declare a queue", slog.Any("error", err.Error()))
		return
	}

	queueName = cfg.QueueNotifications
	_, err = ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Error("Failed to declare a queue", slog.Any("error", err.Error()))
		return
	}

	handler.NewGRPCHandler(cfg, grpcServer, storage, log, ch)
	if err := grpcServer.Serve(l); err != nil {
		log.Error("Error serving gRPC server for PlacesService", slog.String("address", cfg.LocalAddress), slog.String("error", err.Error()))
	}
}
