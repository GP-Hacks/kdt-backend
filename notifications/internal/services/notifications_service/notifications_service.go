package notification_service

import (
	"context"

	"github.com/GP-Hacks/kdt2024-notifications/internal/models"
)

type (
	ITokensRepository interface {
		GetTokensByUserId(ctx context.Context, userId string) ([]string, error)
		AddUserToken(ctx context.Context, userId string, token string) error
	}

	INotificationsRepository interface {
		SendNotifications(ctx context.Context, notification *models.Notification, token ...string) error
	}

	NotificationsService struct {
		tokensRepository       ITokensRepository
		noificationsRepository INotificationsRepository
	}
)

func NewNotificationsService(tokensRepository ITokensRepository, notificationsRepository INotificationsRepository) *NotificationsService {
	return &NotificationsService{
		tokensRepository:       tokensRepository,
		noificationsRepository: notificationsRepository,
	}
}
