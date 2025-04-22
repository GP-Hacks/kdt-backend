package service_provider

import "github.com/GP-Hacks/kdt2024-notifications/internal/controllers/rabbitmq"

func (s *ServiceProvider) NotificationsController() *rabbitmq.NotificationsController {
	if s.notificationsController == nil {
		s.notificationsController = rabbitmq.NewNotificationsController(s.RabbitmqConnection(), s.NotificationsService())
	}

	return s.notificationsController
}
