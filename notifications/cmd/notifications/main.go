package main

import (
	"context"
	"encoding/json"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"flag"
	"github.com/GP-Hack/kdt2024-commons/prettylogger"
	"github.com/GP-Hacks/kdt2024-notifications/config"
	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/api/option"
	"log/slog"
	"time"
)

type NotificationMessage struct {
	UserID  string    `json:"user_id"`
	Header  string    `json:"header"`
	Content string    `json:"content"`
	Time    time.Time `json:"time"`
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

	clientOptions := options.Client().ApplyURI(path)
	mongoClient, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Error("Failed to connect to MongoDB", slog.String("error", err.Error()))
		return
	}
	defer mongoClient.Disconnect(context.Background())

	collection := mongoClient.Database(cfg.MongoDBName).Collection(cfg.MongoDBCollection)

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

	opt := option.WithCredentialsFile(cfg.FirebaseCfg)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Error("Error initializing FirebaseApp", slog.String("error", err.Error()))

	}
	client, err := app.Messaging(context.Background())
	if err != nil {
		log.Error("Error getting Messaging client", slog.String("error", err.Error()))
	}

	for msg := range msgs {
		var notification NotificationMessage
		if err := json.Unmarshal(msg.Body, &notification); err != nil {
			log.Error("Failed to unmarshal message", slog.String("error", err.Error()))
			continue
		}

		if notification.Header == "" || notification.Content == "" || notification.UserID == "" {
			log.Warn("Invalid notification message", slog.String("error", err.Error()))
			continue
		}

		filter := bson.M{"user_id": notification.UserID}
		var userTokens struct {
			Tokens []string `bson:"tokens"`
		}
		err = collection.FindOne(context.Background(), filter).Decode(&userTokens)
		if err != nil {
			log.Warn("Failed to find user tokens", slog.String("error", err.Error()))
			continue
		}

		for _, token := range userTokens.Tokens {
			if err := sendNotification(token, notification.Header, notification.Content, log, client); err != nil {
				log.Error("Failed to send notification", slog.String("error", err.Error()))
			}
		}
	}
}

func sendNotification(token, header, content string, log *slog.Logger, client *messaging.Client) error {
	log.Debug("Sending notification to token with content", slog.String("token", token), slog.String("header", header), slog.String("content", content))

	message := &messaging.Message{
		Token: token,
		Data: map[string]string{
			"title":   header,
			"content": content,
		},
	}

	_, err := client.Send(context.Background(), message)
	if err != nil {
		log.Error("Error sending message", slog.String("error", err.Error()))
	}

	log.Debug("Successfully sent message")
	return nil
}
