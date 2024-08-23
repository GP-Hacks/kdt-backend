package config

import (
	"os"
)

type Config struct {
	Env          string
	Address      string
	RedisAddress string
}

func MustLoad() *Config {
	return &Config{
		Env:          "local",
		Address:      os.Getenv("SERVICE_ADDRESS"),
		RedisAddress: os.Getenv("REDIS_ADDRESS"),
	}
}
