package config

import (
	"os"
)

type Config struct {
	Env                       string
	Address                   string
	RabbitMQAddress           string
	QueueName                 string
	MongoDBName               string
	MongoDBCollection         string
	MongoDBPath               string
	FirebaseProjectId         string
	FirebasePrivateKeyId      string
	FirebasePrivateKey        string
	FirebaseClientEmail       string
	FirebaseClientId          string
	FirebaseClientX509CertUrl string
}

var Cfg Config

func MustLoad() *Config {
	Cfg = Config{
		Env:                       "local",
		Address:                   os.Getenv("SERVICE_ADDRESS"),
		RabbitMQAddress:           os.Getenv("RABBITMQ_ADDRESS"),
		QueueName:                 os.Getenv("QUEUE_NAME"),
		MongoDBName:               os.Getenv("MONGODB_NAME"),
		MongoDBCollection:         os.Getenv("MONGODB_COLLECTION"),
		MongoDBPath:               os.Getenv("MONGODB_PATH"),
		FirebaseProjectId:         os.Getenv("FIREBASE_PROJECT_ID"),
		FirebasePrivateKeyId:      os.Getenv("FIREBASE_PRIVATE_KEY_ID"),
		FirebasePrivateKey:        os.Getenv("FIREBASE_PRIVATE_KEY"),
		FirebaseClientEmail:       os.Getenv("FIREBASE_CLIENT_EMAIL"),
		FirebaseClientId:          os.Getenv("FIREBASE_CLIENT_ID"),
		FirebaseClientX509CertUrl: os.Getenv("FIREBASE_CLIENT_X509_CERT_URL"),
	}

	return &Cfg
}
