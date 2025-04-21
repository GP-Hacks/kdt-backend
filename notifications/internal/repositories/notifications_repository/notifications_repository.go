package notifications_repository

import "firebase.google.com/go/messaging"

type NotificationsRepository struct {
	client *messaging.Client
}

func NewNotificationsRepository(client *messaging.Client) *NotificationsRepository {
	return &NotificationsRepository{
		client: client,
	}
}
