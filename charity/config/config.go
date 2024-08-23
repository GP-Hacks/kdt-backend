package config

import "os"

type Config struct {
	Env             string
	Address         string
	PostgresAddress string
	RabbitMQAddress string
	QueueName       string
}

func MustLoad() *Config {
	return &Config{
		Env:             "local",
		Address:         os.Getenv("SERVICE_ADDRESS"),
		PostgresAddress: os.Getenv("POSTGRES_ADDRESS"),
		RabbitMQAddress: os.Getenv("RABBITMQ_ADDRESS"),
		QueueName:       os.Getenv("QUEUE_NAME"),
	}
}
