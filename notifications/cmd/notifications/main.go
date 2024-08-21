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
	flag.StringVar(&path, "path", "", "mongoDBUri")
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
	log.Info("MongoDB connected")

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
	log.Info("RabbitMQ connected")

	opt := option.WithCredentialsFile(cfg.FirebaseCfg)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Error("Error initializing FirebaseApp", slog.String("error", err.Error()))
		return
	}
	client, err := app.Messaging(context.Background())
	if err != nil {
		log.Error("Error getting Messaging client", slog.String("error", err.Error()))
		return
	}
	log.Info("Firebase connected")

	for msg := range msgs {
		var notification NotificationMessage
		if err := json.Unmarshal(msg.Body, &notification); err != nil {
			log.Error("Failed to unmarshal message", slog.String("error", err.Error()))
			continue
		}

		if notification.Header == "" || notification.Content == "" || notification.UserID == "" {
			log.Warn("Invalid notification message")
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
		locationMSK := time.FixedZone("MSK", 3*60*60)
		notificationTime := time.Date(
			notification.Time.Year(), notification.Time.Month(), notification.Time.Day(),
			notification.Time.Hour(), notification.Time.Minute(), notification.Time.Second(),
			notification.Time.Nanosecond(), locationMSK)
		notification.Time = notificationTime
		delay := time.Until(notificationTime)
		if delay < 0 {
			log.Warn("Notification time is in the past, sending immediately")
			delay = 0
		}

		for _, token := range userTokens.Tokens {
			go func(token string) {
				time.AfterFunc(delay, func() {
					_ = sendNotification(token, notification.Header, notification.Content, log, client)
				})
			}(token)
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
		log.Warn("Error sending message", slog.String("error", err.Error()))
		return err
	}

	log.Debug("Successfully sent message")
	return nil
}
