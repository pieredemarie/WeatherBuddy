package service

import (
	"WeatherBuddy/registration-service/internal/model"
	"context"
)

type DueUsersRepo interface {
	DueUsers(ctx context.Context) ([]model.User, error)
}

type JobPublisher interface {
	Publish(ctx, context.Context, j)
}
