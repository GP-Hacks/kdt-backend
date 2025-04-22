package service_provider

import notification_service "github.com/GP-Hacks/kdt2024-notifications/internal/services/notifications_service"

func (s *ServiceProvider) NotificationsService() *notification_service.NotificationsService {
	if s.notificationsService == nil {
		s.notificationsService = notification_service.NewNotificationsService(s.TokensRepository(), s.NotificationsRepository())
	}

	return s.notificationsService
}
