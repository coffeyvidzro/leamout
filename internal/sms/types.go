package sms

import "github.com/google/uuid"

type Config struct {
	DefaultFrom string
}

type Message struct {
	UserID uuid.UUID

	To      string
	Content string
	From    string

	// Reference should be unique per business action.
	// Example: dunning_sms:<dunning_attempt_id>
	Reference string

	Metadata map[string]any
}
