package app

import (
	"log"
	"time"

	"registration-service/internal/config"
	"registration-service/internal/geocoding"
	tghandler "registration-service/internal/handlers/telegram"
)

type stubUserStore struct{}

func (stubUserStore) Save(u tghandler.RegisteredUser) error {
	log.Printf("[stub store] saved: %+v", u)
	return nil
}

func Run() {
	cfg := config.MustLoad()

	geocoder := geocoding.WithRetry(
		geocoding.NewOpenMeteoGeocoder(),
		3,                    // maxAttempts
		500*time.Millisecond, // baseDelay: 500ms, 1s, 2s
	)

	handler, err := tghandler.New(cfg.BotToken, stubUserStore{}, geocoder)
	if err != nil {
		log.Fatal(err)
	}

	handler.Register()

	log.Println("telegram bot started")

	handler.Start()
}
