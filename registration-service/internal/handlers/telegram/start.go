package telegram

import "gopkg.in/telebot.v4"

func (h *Handler) Start() {
	h.bot.Start()
}

func (h *Handler) start(c telebot.Context) error {
	return c.Send("WeatherBuddy started")
}
