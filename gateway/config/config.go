package config

import (
	"os"
	"time"
)

type Config struct {
	Env               string
	LocalAddress      string
	Address           string
	ChatAddress       string
	PlacesAddress     string
	CharityAddress    string
	VotesAddress      string
	Timeout           time.Duration
	IdleTimeout       time.Duration
	MongoDBName       string
	MongoDBCollection string
	MongoDBPath       string
}

func MustLoad() *Config {
	return &Config{
		Env:               "local",
		Address:           os.Getenv("SERVICE_ADDRESS"),
		LocalAddress:      os.Getenv("LOCAL_ADDRESS"),
		ChatAddress:       os.Getenv("CHAT_SERVICE_ADDRESS"),
		PlacesAddress:     os.Getenv("PLACES_SERVICE_ADDRESS"),
		CharityAddress:    os.Getenv("CHARITY_SERVICE_ADDRESS"),
		VotesAddress:      os.Getenv("VOTES_SERVICE_ADDRESS"),
		Timeout:           time.Second * 15,
		IdleTimeout:       time.Second * 60,
		MongoDBName:       os.Getenv("MONGODB_NAME"),
		MongoDBCollection: os.Getenv("MONGODB_COLLECTION"),
		MongoDBPath:       os.Getenv("MONGODB_PATH"),
	}
}
