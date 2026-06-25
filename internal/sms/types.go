package sms

import "github.com/google/uuid"

type Config struct {
	DefaultFrom string
}

type Message struct {
	UserID    uuid.UUID
	To        string
	Content   string
	From      string
	Reference string
	Metadata  map[string]any
}
