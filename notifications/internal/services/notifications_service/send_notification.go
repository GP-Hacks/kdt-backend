package notification_service

import (
	"context"
	"time"

	"github.com/GP-Hacks/kdt2024-notifications/internal/models"
)

func (s *NotificationsService) SendNotifications(ctx context.Context, notification *models.Notification) {
	delay := time.Until(notification.Time)
	if delay < 0 {
		delay = 0
	}

	tokens, err := s.tokensRepository.GetTokensByUserId(ctx, notification.UserId)
	if err != nil {
		// TODO: добавить логирование или сбор метрик надо
	}

	for _, token := range tokens {
		go func(token string) {
			time.AfterFunc(delay, func() {
				s.noificationsRepository.SendNotifications(ctx, notification, token)
			})
		}(token)
	}
}
