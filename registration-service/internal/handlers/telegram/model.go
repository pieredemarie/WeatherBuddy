package telegram

import (
	"context"
	"registration-service/internal/model"
	"time"
)

const inactivityTimeout = 15 * time.Second

type UserStore interface {
	Save(ctx context.Context, u model.User) error
}

type registrationStep int

const (
	stepAwaitingCity registrationStep = iota
	stepAwaitingTime
)

type registrationState struct {
	step       registrationStep
	city       string
	latitude   float64
	longitude  float64
	timezone   string
	notifyTime time.Time

	cancelTimeout context.CancelFunc
}
