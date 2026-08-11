package telegram

import "gopkg.in/telebot.v4"

func (h *Handler) register(c telebot.Context) error {
	if c.Sender() == nil {
		return nil
	}
	userID := c.Sender().ID
	state := &registrationState{step: stepAwaitingCity}
	h.states.set(userID, state)
	h.armInactivityTimer(userID, state)
	return c.Send("Введите ваш город:")
}
