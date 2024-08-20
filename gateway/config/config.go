package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log"
	"os"
	"time"
)

type Config struct {
	Env               string        `yaml:"env" env-required:"true"`
	Address           string        `yaml:"address" env-required:"true"`
	ChatAddress       string        `yaml:"chat_address" env-required:"true"`
	PlacesAddress     string        `yaml:"places_address" env-required:"true"`
	Timeout           time.Duration `yaml:"timeout" env-required:"true"`
	IdleTimeout       time.Duration `yaml:"idle_timeout" env-required:"true"`
	MongoDBName       string        `yaml:"mongodb_name" env-required:"true"`
	MongoDBCollection string        `yaml:"mongodb_collection" env-required:"true"`
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		// TODO: move to env
		configPath = "gateway/config/config.yaml"
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
