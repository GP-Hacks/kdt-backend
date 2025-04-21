package notifications_repository

import (
	"context"

	"firebase.google.com/go/messaging"
	"github.com/GP-Hacks/kdt2024-notifications/internal/models"
)

func (r *NotificationsRepository) SendNotifications(ctx context.Context, notification *models.Notification, tokens ...string) error {
	for _, token := range tokens {
		go func() {
			message := &messaging.Message{
				Token: token,
				Data: map[string]string{
					"title":   notification.Header,
					"content": notification.Content,
				},
			}

			_, err := r.client.Send(context.Background(), message)
			if err != nil {
				// TODO: логи или чет такое надо
			}

		}()
	}

	return nil
}
