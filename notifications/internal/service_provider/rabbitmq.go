package service_provider

import (
	"github.com/GP-Hacks/kdt2024-notifications/config"
	"github.com/streadway/amqp"
)

func (s *ServiceProvider) RabbitmqConnection() *amqp.Connection {
	if s.rabbitmqConnection == nil {
		conn, err := amqp.Dial(config.Cfg.RabbitMQAddress)
		if err != nil {
			panic(err)
		}

		s.rabbitmqConnection = conn
	}

	return s.rabbitmqConnection
}
