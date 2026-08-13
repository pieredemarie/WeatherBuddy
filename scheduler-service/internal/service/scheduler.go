package service

import (
	"context"
	"fmt"
	"log"
	"scheduler-service/internal/model"
	"scheduler-service/internal/queue"
)

type DueUsersRepo interface {
	DueUsers(ctx context.Context) ([]model.User, error)
}

type JobPublisher interface {
	Publish(ctx context.Context, job queue.NotificationJob) error
}

type SchedulerService struct {
	repo      DueUsersRepo
	publisher JobPublisher
}

func NewSchedulerService(repo DueUsersRepo, publisher JobPublisher) *SchedulerService {
	return &SchedulerService{repo: repo, publisher: publisher}
}

func (s *SchedulerService) Tick(ctx context.Context) error {
	users, err := s.repo.DueUsers(ctx)
	if err != nil {
		return fmt.Errorf("fetch due users %w", err)
	}

	for _, u := range users {
		job := queue.NotificationJob{
			UserID:       u.ID,
			ContactType:  string(u.ContactType),
			ContactValue: u.ContactValue,
			City:         u.City,
			Latitude:     u.Latitude,
			Longitude:    u.Longitude,
			Timezone:     u.Timezone,
		}

		if err := s.publisher.Publish(ctx, job); err != nil {
			log.Printf("scheduler: publish job for user %d failed: %v", u.ID, err)
			continue
		}
	}

	return nil
}
