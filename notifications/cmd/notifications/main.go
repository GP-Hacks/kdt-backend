package main

import (
	"context"
	"encoding/json"
	"errors"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"github.com/GP-Hacks/kdt2024-commons/prettylogger"
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
	log.Info("Configuration loaded successfully")

	mongoClient, err := setupMongoDB(cfg, log)
	if err != nil {
		log.Error("Failed to setup MongoDB", slog.String("error", err.Error()))
		return
	}
	defer func() {
		if err := mongoClient.Disconnect(context.Background()); err != nil {
			log.Error("Failed to disconnect MongoDB", slog.String("error", err.Error()))
		}
	}()
	log.Info("MongoDB connection established")

	conn, ch, err := setupRabbitMQ(cfg, log)
	if err != nil {
		log.Error("Failed to setup RabbitMQ", slog.String("error", err.Error()))
		return
	}
	defer func() {
		if err := ch.Close(); err != nil {
			log.Error("Failed to close RabbitMQ channel", slog.String("error", err.Error()))
		}
		if err := conn.Close(); err != nil {
			log.Error("Failed to close RabbitMQ connection", slog.String("error", err.Error()))
		}
	}()
	log.Info("RabbitMQ connection established")

	client, err := setupFirebase(cfg, log)
	if err != nil {
		log.Error("Failed to setup Firebase", slog.String("error", err.Error()))
		return
	}
	log.Info("Firebase setup successfully")

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
		log.Error("Failed to register RabbitMQ consumer", slog.String("error", err.Error()))
		return
	}
	log.Info("RabbitMQ consumer registered successfully", slog.String("queue", cfg.QueueName))

	processMessages(msgs, mongoClient, cfg, log, client)
}

func setupMongoDB(cfg *config.Config, log *slog.Logger) (*mongo.Client, error) {
	log.Info("Connecting to MongoDB", slog.String("uri", cfg.MongoDBPath))
	clientOptions := options.Client().ApplyURI(cfg.MongoDBPath)
	mongoClient, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Error("MongoDB connection failed", slog.String("uri", cfg.MongoDBPath), slog.String("error", err.Error()))
		return nil, err
	}
	return mongoClient, nil
}

func setupRabbitMQ(cfg *config.Config, log *slog.Logger) (*amqp.Connection, *amqp.Channel, error) {
	log.Info("Connecting to RabbitMQ", slog.String("uri", cfg.RabbitMQAddress))
	conn, err := amqp.Dial(cfg.RabbitMQAddress)
	if err != nil {
		log.Error("RabbitMQ connection failed", slog.String("uri", cfg.RabbitMQAddress), slog.String("error", err.Error()))
		return nil, nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Error("Failed to open RabbitMQ channel", slog.String("error", err.Error()))
		return nil, nil, err
	}
	return conn, ch, nil
}

func setupFirebase(cfg *config.Config, log *slog.Logger) (*messaging.Client, error) {
	log.Info("Initializing FirebaseApp")
	creds := map[string]string{
		"type":                        "service_account",
		"project_id":                  cfg.FirebaseProjectId,
		"private_key_id":              cfg.FirebasePrivateKeyId,
		"private_key":                 cfg.FirebasePrivateKey,
		"client_email":                cfg.FirebaseClientEmail,
		"client_id":                   cfg.FirebaseClientId,
		"auth_uri":                    "https://accounts.google.com/o/oauth2/auth",
		"token_uri":                   "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url":        cfg.FirebaseClientX509CertUrl,
		"universe_domain":             "googleapis.com",
	}

	credentialsJSON, err := json.Marshal(creds)
	if err != nil {
		log.Error("Failed to marshal Firebase credentials", slog.String("error", err.Error()))
		return nil, err
	}

	opt := option.WithCredentialsJSON(credentialsJSON)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Error("Failed to initialize FirebaseApp", slog.String("error", err.Error()))
		return nil, err
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		log.Error("Failed to initialize Firebase Messaging client", slog.String("error", err.Error()))
		return nil, err
	}
	return client, nil
}

func processMessages(msgs <-chan amqp.Delivery, mongoClient *mongo.Client, cfg *config.Config, log *slog.Logger, client *messaging.Client) {
	collection := mongoClient.Database(cfg.MongoDBName).Collection(cfg.MongoDBCollection)

	for msg := range msgs {
		var notification NotificationMessage
		if err := json.Unmarshal(msg.Body, &notification); err != nil {
			log.Error("Failed to unmarshal RabbitMQ message", slog.String("error", err.Error()), slog.Any("body", msg.Body))
			continue
		}
		log.Info("Received notification message", slog.Any("notification", notification))

		if err := validateNotification(notification); err != nil {
			log.Warn("Invalid notification message", slog.String("error", err.Error()))
			continue
		}

		userTokens, err := fetchUserTokens(collection, notification.UserID, log)
		if err != nil {
			log.Warn("Failed to fetch user tokens", slog.String("error", err.Error()), slog.String("user_id", notification.UserID))
			continue
		}

		notification.Time = adjustNotificationTime(notification.Time, log)
		sendNotifications(userTokens, notification, log, client)
	}
}

func validateNotification(notification NotificationMessage) error {
	if notification.Header == "" || notification.Content == "" || notification.UserID == "" {
		return errors.New("missing required fields")
	}
	return nil
}

func fetchUserTokens(collection *mongo.Collection, userID string, log *slog.Logger) ([]string, error) {
	log.Debug("Fetching user tokens from MongoDB", slog.String("user_id", userID))
	filter := bson.M{"user_id": userID}
	var userTokens struct {
		Tokens []string `bson:"tokens"`
	}

	err := collection.FindOne(context.Background(), filter).Decode(&userTokens)
	if err != nil {
		log.Warn("Failed to find user tokens in MongoDB", slog.String("user_id", userID), slog.String("error", err.Error()))
		return nil, err
	}
	return userTokens.Tokens, nil
}

func adjustNotificationTime(notificationTime time.Time, log *slog.Logger) time.Time {
	locationMSK := time.FixedZone("MSK", 3*60*60)
	adjustedTime := time.Date(
		notificationTime.Year(), notificationTime.Month(), notificationTime.Day(),
		notificationTime.Hour(), notificationTime.Minute(), notificationTime.Second(),
		notificationTime.Nanosecond(), locationMSK)
	log.Info("Adjusted notification time", slog.Time("original_time", notificationTime), slog.Time("adjusted_time", adjustedTime))

	return adjustedTime
}

func sendNotifications(tokens []string, notification NotificationMessage, log *slog.Logger, client *messaging.Client) {
	delay := time.Until(notification.Time)
	if delay < 0 {
		log.Warn("Notification time is in the past, sending immediately", slog.Time("notification_time", notification.Time))
		delay = 0
	}

	for _, token := range tokens {
		go func(token string) {
			time.AfterFunc(delay, func() {
				if err := sendNotification(token, notification.Header, notification.Content, log, client); err != nil {
					log.Warn("Failed to send notification", slog.String("token", token), slog.String("error", err.Error()))
				} else {
					log.Info("Notification sent successfully", slog.String("token", token), slog.String("header", notification.Header))
				}
			})
		}(token)
	}
}

func sendNotification(token, header, content string, log *slog.Logger, client *messaging.Client) error {
	log.Debug("Sending notification", slog.String("token", token), slog.String("header", header), slog.String("content", content))

	message := &messaging.Message{
		Token: token,
		Data: map[string]string{
			"title":   header,
			"content": content,
		},
	}

	_, err := client.Send(context.Background(), message)
	if err != nil {
		log.Error("Failed to send notification", slog.String("token", token), slog.String("error", err.Error()))
		return err
	}

	log.Info("Notification sent successfully", slog.String("token", token))
	return nil
}
