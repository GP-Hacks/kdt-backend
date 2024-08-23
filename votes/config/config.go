package config

import "os"

type Config struct {
	Env             string
	Address         string
	PostgresAddress string
	LocalAddress    string
}

func MustLoad() *Config {
	return &Config{
		Env:             "local",
		Address:         os.Getenv("SERVICE_ADDRESS"),
		PostgresAddress: os.Getenv("POSTGRES_ADDRESS"),
		LocalAddress:    os.Getenv("LOCAL_ADDRESS"),
	}
}
