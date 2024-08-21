package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/GP-Hack/kdt2024-commons/prettylogger"
	"github.com/GP-Hacks/kdt2024-purchases/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/streadway/amqp"
	"log/slog"
	"time"
)

type PurchaseMessage struct {
	UserToken    string    `json:"user_token"`
	PlaceID      int       `json:"place_id"`
	EventTime    time.Time `json:"event_time"`
	PurchaseTime time.Time `json:"purchase_time"`
	Cost         int       `json:"cost"`
}

func main() {
	cfg := config.MustLoad()
	log := prettylogger.SetupLogger(cfg.Env)
	log.Info("Configuration loaded")
	log.Info("Logger loaded")

	var path string
	flag.StringVar(&path, "path", "", "postgres://username:password@host:port/dbname")
	flag.Parse()
	if path == "" {
		log.Error("No storage_path provided")
		return
	}

	dbpool, err := pgxpool.New(context.Background(), path+"?sslmode=disable")
	if err != nil {
		log.Error("Failed to connect to Postgres", slog.String("error", err.Error()))
		return
	}
	defer dbpool.Close()
	log.Info("PostgreSQL connected")

	_, err = dbpool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS ticket_purchases (
			user_token TEXT,
			place_id INT REFERENCES places(id) ON DELETE CASCADE,
			event_time TIMESTAMP,
			purchase_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			cost INT
		)
	`)
	if err != nil {
		log.Error("Failed to create table", slog.String("error", err.Error()))
		return
	}

	conn, err := amqp.Dial(cfg.RabbitMQAddress)
	if err != nil {
		log.Error("Failed to connect to RabbitMQ", slog.String("error", err.Error()))
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Error("Failed to open a channel", slog.String("error", err.Error()))
		return
	}
	defer ch.Close()

	msgs, err := ch.Consume(
		cfg.QueueName,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Error("Failed to register a consumer", slog.String("error", err.Error()))
		return
	}

	log.Info("RabbitMQ connected")

	for msg := range msgs {
		var dbmsg PurchaseMessage
		if err := json.Unmarshal(msg.Body, &dbmsg); err != nil {
			log.Error("Failed to unmarshal message", slog.String("error", err.Error()))
			continue
		}

		if dbmsg.UserToken == "" || dbmsg.PlaceID == 0 || dbmsg.EventTime.IsZero() || dbmsg.Cost == 0 {
			log.Warn("Invalid purchase message")
			continue
		}

		_, err := dbpool.Exec(context.Background(), `INSERT INTO ticket_purchases(user_token, place_id, event_time, purchase_time, cost) VALUES ($1, $2, $3, $4, $5)`, dbmsg.UserToken, dbmsg.PlaceID, dbmsg.EventTime, dbmsg.PurchaseTime, dbmsg.Cost)
		if err != nil {
			log.Error("Failed to save purchase to Postgres", slog.String("error", err.Error()))
			continue
		}
		log.Debug("Saved ticket purchase")
	}
}
