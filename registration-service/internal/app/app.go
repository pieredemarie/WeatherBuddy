package app

import (
	"context"
	"log"
	"registration-service/internal/repository/postgres"
	"time"

	"registration-service/internal/config"
	"registration-service/internal/geocoding"
	tghandler "registration-service/internal/handlers/telegram"
)

func Run() {
	cfg := config.MustLoad()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := postgres.NewDB(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	store := postgres.NewPostgresRepo(db)

	geocoder := geocoding.WithRetry(
		geocoding.NewOpenMeteoGeocoder(),
		3,                    // maxAttempts
		500*time.Millisecond, // baseDelay: 500ms, 1s, 2s
	)

	handler, err := tghandler.New(cfg.BotToken, store, geocoder)
	if err != nil {
		log.Fatal(err)
	}

	handler.Register()

	log.Println("telegram bot started")

	handler.Start()
}
