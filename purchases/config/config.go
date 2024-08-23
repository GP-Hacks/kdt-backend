package config

import (
	"os"
)

type Config struct {
	Env             string
	Address         string
	RabbitMQAddress string
	QueueName       string
	PostgresAddress string
	LocalAddress    string
}

func MustLoad() *Config {
	return &Config{
		Env:             "local",
		Address:         os.Getenv("SERVICE_ADDRESS"),
		RabbitMQAddress: os.Getenv("RABBITMQ_ADDRESS"),
		QueueName:       os.Getenv("QUEUE_NAME"),
		PostgresAddress: os.Getenv("POSTGRES_ADDRESS"),
		LocalAddress:    os.Getenv("LOCAL_ADDRESS"),
	}
}
