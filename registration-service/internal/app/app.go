package app

import (
	"log"
	"registration-service/internal/config"
	tghandler "registration-service/internal/handlers/telegram"
)

func Run() {
	cfg := config.MustLoad()

	handler, err := tghandler.New(cfg.BotToken)
	if err != nil {
		log.Fatal(err)
	}

	handler.Register()

	log.Println("telegram bot started")

	handler.Start()

}
