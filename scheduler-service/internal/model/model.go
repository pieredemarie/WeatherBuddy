package model

import "time"

type ContactType string

const (
	ContactTelegram ContactType = "Telegram"
	ContactEmail    ContactType = "Email"
)

type User struct {
	ID           int
	ContactType  ContactType
	ContactValue string

	City       string
	Latitude   float64
	Longitude  float64
	Timezone   string
	NotifyTime time.Time

	Active    bool
	CreatedAt time.Time
}
