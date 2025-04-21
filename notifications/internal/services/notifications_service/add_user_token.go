package notification_service

import "context"

func (s *NotificationsService) AddUserToken(ctx context.Context, userId, token string) error {
	return s.tokensRepository.AddUserToken(ctx, userId, token)
}
