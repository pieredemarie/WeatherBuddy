package service

import "context"

type UserRepository interface {
	Create(ctx context.Context)
}
