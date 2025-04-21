package notification_service

import (
	"context"

	"github.com/GP-Hacks/kdt2024-notifications/internal/models"
)

func (s *NotificationsService) SendNotification(ctx context.Context, notification *models.Notification, userIds ...string) {
	for _, userId := range userIds {
		go func(userId string) {
			tokens, err := s.tokensRepository.GetTokensByUserId(ctx, userId)
			if err != nil {
				// TODO: добавить логирование или сбор метрик надо
			}

			s.noificationsRepository.SendNotification(ctx, notification, tokens...)
		}(userId)
	}
}
