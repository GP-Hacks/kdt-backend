package rabbitmq

import (
	"context"
	"encoding/json"

	"github.com/GP-Hacks/kdt2024-notifications/config"
	"github.com/GP-Hacks/kdt2024-notifications/internal/models"
	notification_service "github.com/GP-Hacks/kdt2024-notifications/internal/services/notifications_service"
	"github.com/streadway/amqp"
)

type NotificationsController struct {
	connection           *amqp.Connection
	notificationsService *notification_service.NotificationsService
}

func NewNotificationsController(conn *amqp.Connection, service *notification_service.NotificationsService) *NotificationsController {
	return &NotificationsController{
		connection:           conn,
		notificationsService: service,
	}
}

func (c *NotificationsController) StartConsumer() error {
	ch, err := c.connection.Channel()
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(
		config.Cfg.QueueName,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for {
			msg := <-msgs

			var notification models.Notification
			if err := json.Unmarshal(msg.Body, &notification); err != nil {
				continue
			}

			if notification.Header == "" || notification.Content == "" || notification.UserId == "" {
				continue
			}

			c.notificationsService.SendNotifications(
				context.Background(),
				&notification,
			)
		}
	}()

	return nil
}
