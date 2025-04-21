package models

import "time"

type Notification struct {
	Header  string    `json:"header,omitempty"`
	Content string    `json:"content,omitempty"`
	Time    time.Time `json:"time,omitempty"`
	UserId  string    `json:"user_id,omitempty"`
}
