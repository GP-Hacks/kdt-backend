package service_provider

import (
	"github.com/GP-Hacks/kdt2024-notifications/internal/repositories/notifications_repository"
	"github.com/GP-Hacks/kdt2024-notifications/internal/repositories/tokens_repository"
)

func (s *ServiceProvider) NotificationsRepository() *notifications_repository.NotificationsRepository {
	if s.notificationsRepository == nil {
		s.notificationsRepository = notifications_repository.NewNotificationsRepository(s.MessagingClient())
	}

	return s.notificationsRepository
}

func (s *ServiceProvider) TokensRepository() *tokens_repository.TokensRepository {
	if s.tokensRepository == nil {
		s.tokensRepository = tokens_repository.NewTokensRepository(s.MongoCollection())
	}

	return s.tokensRepository
}
