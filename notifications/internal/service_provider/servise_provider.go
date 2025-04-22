package service_provider

import (
	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"github.com/GP-Hacks/kdt2024-notifications/internal/controllers/rabbitmq"
	"github.com/GP-Hacks/kdt2024-notifications/internal/repositories/notifications_repository"
	"github.com/GP-Hacks/kdt2024-notifications/internal/repositories/tokens_repository"
	notification_service "github.com/GP-Hacks/kdt2024-notifications/internal/services/notifications_service"
	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/mongo"
)

// ServiceProvider struct  
// Struct for provide service objects
type ServiceProvider struct {
	notificationsController *rabbitmq.NotificationsController
	notificationsService    *notification_service.NotificationsService
	notificationsRepository *notifications_repository.NotificationsRepository
	tokensRepository        *tokens_repository.TokensRepository
	mongoCollection         *mongo.Collection
	mongoClient             *mongo.Client
	firebaseApp             *firebase.App
	messagingClient         *messaging.Client
	rabbitmqConnection      *amqp.Connection
}

func NewServiceProvider() *ServiceProvider {
	return &ServiceProvider{}
}
