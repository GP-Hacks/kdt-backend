package service_provider

import (
	"context"
	"encoding/json"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"github.com/GP-Hacks/kdt2024-notifications/config"
	"google.golang.org/api/option"
)

func (s *ServiceProvider) FirebaseApp() *firebase.App {
	if s.firebaseApp == nil {
		creds := map[string]string{
			"type":                        "service_account",
			"project_id":                  config.Cfg.FirebaseProjectId,
			"private_key_id":              config.Cfg.FirebasePrivateKeyId,
			"private_key":                 config.Cfg.FirebasePrivateKey,
			"client_email":                config.Cfg.FirebaseClientEmail,
			"client_id":                   config.Cfg.FirebaseClientId,
			"auth_uri":                    "https://accounts.google.com/o/oauth2/auth",
			"token_uri":                   "https://oauth2.googleapis.com/token",
			"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
			"client_x509_cert_url":        config.Cfg.FirebaseClientX509CertUrl,
			"universe_domain":             "googleapis.com",
		}

		credentialsJSON, err := json.Marshal(creds)
		if err != nil {
			panic(err)
		}

		opt := option.WithCredentialsJSON(credentialsJSON)
		app, err := firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			panic(err)
		}

		s.firebaseApp = app
	}

	return s.firebaseApp
}

func (s *ServiceProvider) MessagingClient() *messaging.Client {
	if s.messagingClient == nil {
		client, err := s.FirebaseApp().Messaging(context.Background())
		if err != nil {
			panic(err)
		}

		s.messagingClient = client
	}

	return s.messagingClient
}
