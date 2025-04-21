package notification_service

import (
	"context"

	"github.com/GP-Hacks/kdt2024-notifications/internal/models"
)

func (s *NotificationsService) SendNotification(ctx context.Context, notification *models.Notification, userIds ...string) {
}
