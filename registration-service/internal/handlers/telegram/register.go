package telegram

import "gopkg.in/telebot.v4"

func (h *Handler) register(c telebot.Context) error {
	if c.Sender() == nil {
		return nil
	}
	userID := c.Sender().ID
	h.states.set(userID, &registrationState{step: stepAwaitingCity})
	return c.Send("Введите ваш город:")
}
