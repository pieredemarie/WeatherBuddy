package main

import (
	"context"
	"log"
	"scheduler-service/internal/config"
	"scheduler-service/internal/queue"
	"scheduler-service/internal/repository/postgres"
	"scheduler-service/internal/service"
	"time"
)

func main() {
	cfg := config.MustLoad()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	db, err := postgres.NewDB(ctx, cfg.PostgresDSN)
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	repo := postgres.NewSchedulerRepo(db)

	producer := queue.NewProducer(cfg.KafkaBrokers, queue.NotificationJobsTopic)
	defer producer.Close()

	schedulerSvc := service.NewSchedulerService(repo, producer)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	log.Println("scheduler service started")

	for range ticker.C {
		tickCtx, tickCancel := context.WithTimeout(context.Background(), 30*time.Second)

		if err := schedulerSvc.Tick(tickCtx); err != nil {
			log.Println("scheduler service tick failed: %v", err)
		}
		tickCancel()
	}
}
