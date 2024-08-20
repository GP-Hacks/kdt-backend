package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log"
	"os"
)

type Config struct {
	Env               string `yaml:"env" env-required:"true"`
	Address           string `yaml:"address" env-required:"true"`
	RabbitMQAddress   string `yaml:"rabbitmq_address" env-required:"true"`
	QueueName         string `yaml:"queue_name" env-required:"true"`
	FirebaseCfg       string `yaml:"firebase_cfg" env-required:"true"`
	MongoDBName       string `yaml:"mongodb_name" env-required:"true"`
	MongoDBCollection string `yaml:"mongodb_collection" env-required:"true"`
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		// TODO: move to env
		configPath = "notifications/config/config.yaml"
		//log.Fatal("CONFIG_PATH environment variable is not set")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("%s: CONFIG_PATH does not exist", configPath)
	}

	var config Config
	if err := cleanenv.ReadConfig(configPath, &config); err != nil {
		log.Fatalf("%s: CONFIG_PATH read error: %v", configPath, err)
	}

	return &config
}
