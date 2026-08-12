package telegram

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/telebot.v4"

	"registration-service/internal/geocoding"
	"registration-service/internal/model"
)

type RegistrationService interface {
	ResolveCity(ctx context.Context, city string) (geocoding.GeoLocation, error)
	ParseNotifyTime(text string) (time.Time, error)
	Register(ctx context.Context, contactType model.ContactType, contactValue string, loc geocoding.GeoLocation, notifyTime time.Time) (model.User, error)
}

const inactivityTimeout = 5 * time.Second

type registrationStep int

const (
	stepAwaitingCity registrationStep = iota
	stepAwaitingTime
)

type registrationState struct {
	step       registrationStep
	location   geocoding.GeoLocation
	notifyTime time.Time

	cancelTimeout context.CancelFunc
}

type stateStore struct {
	mu     sync.Mutex
	states map[int64]*registrationState
}

func newStateStore() *stateStore {
	return &stateStore{states: make(map[int64]*registrationState)}
}

func (s *stateStore) get(userID int64) (*registrationState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.states[userID]
	return state, ok
}

func (s *stateStore) set(userID int64, state *registrationState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[userID] = state
}

func (s *stateStore) delete(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.states, userID)
}

func (s *stateStore) deleteIfSame(userID int64, state *registrationState) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.states[userID]
	if exists && current == state {
		delete(s.states, userID)
		return true
	}
	return false
}

type Handler struct {
	bot     *telebot.Bot
	service RegistrationService
	states  *stateStore
}

func New(token string, service RegistrationService) (*Handler, error) {
	bot, err := telebot.NewBot(telebot.Settings{
		Token: token,
		Poller: &telebot.LongPoller{
			Timeout: 10 * time.Second,
		},
	})
	if err != nil {
		return nil, err
	}
	return &Handler{
		bot:     bot,
		service: service,
		states:  newStateStore(),
	}, nil
}

func (h *Handler) Register() {
	h.bot.Handle("/start", h.start)
	h.bot.Handle("/register", h.register)
	h.bot.Handle("/cancel", h.cancel)
	h.bot.Handle(telebot.OnText, h.handleRegistrationMessage)
}

func (h *Handler) Start() {
	h.bot.Start()
}

func (h *Handler) start(c telebot.Context) error {
	return c.Send("WeatherBuddy started")
}

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

func (h *Handler) cancel(c telebot.Context) error {
	if c.Sender() == nil {
		return nil
	}
	userID := c.Sender().ID
	state, exists := h.states.get(userID)
	if !exists {
		return c.Send("Сейчас нет активной регистрации.")
	}
	if state.cancelTimeout != nil {
		state.cancelTimeout()
	}
	h.states.delete(userID)
	return c.Send("Регистрация отменена. Начать заново — /register")
}

func (h *Handler) handleRegistrationMessage(c telebot.Context) error {
	if c.Sender() == nil {
		return nil
	}
	userID := c.Sender().ID

	state, exists := h.states.get(userID)
	if !exists {
		return nil
	}

	var stepErr error
	switch state.step {
	case stepAwaitingCity:
		stepErr = h.handleCityStep(c, userID, state)
	case stepAwaitingTime:
		stepErr = h.handleTimeStep(c, userID, state)
	default:
		h.states.delete(userID)
		return c.Send("Что-то пошло не так, начните заново: /register")
	}

	if current, stillActive := h.states.get(userID); stillActive && current == state {
		h.armInactivityTimer(userID, current)
	}

	return stepErr
}

func (h *Handler) armInactivityTimer(userID int64, state *registrationState) {
	if state.cancelTimeout != nil {
		state.cancelTimeout()
	}
	ctx, cancel := context.WithTimeout(context.Background(), inactivityTimeout)
	state.cancelTimeout = cancel
	go h.watchInactivity(ctx, userID, state)
}

func (h *Handler) watchInactivity(ctx context.Context, userID int64, state *registrationState) {
	<-ctx.Done()
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return
	}
	if h.states.deleteIfSame(userID, state) {
		h.bot.Send(telebot.ChatID(userID), "Вы были неактивны! Регистрация отменена. Начните заново — /register")
	}
}

func (h *Handler) handleCityStep(c telebot.Context, userID int64, state *registrationState) error {
	cityInput := strings.TrimSpace(c.Text())
	if cityInput == "" {
		return c.Send("Город не может быть пустым. Попробуйте ещё раз:")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	loc, err := h.service.ResolveCity(ctx, cityInput)
	switch {
	case errors.Is(err, geocoding.ErrCityNotFound):
		return c.Send("Не смог найти такой город. Проверьте написание и попробуйте ещё раз:")
	case err != nil:
		return c.Send("Не получилось определить часовой пояс для этого города (сервис временно недоступен). Попробуйте ещё раз чуть позже:")
	}

	state.location = loc
	state.step = stepAwaitingTime
	h.states.set(userID, state)

	return c.Send(fmt.Sprintf(
		"Город сохранён: %s (часовой пояс: %s)\n\nТеперь введите время рассылки в формате HH:MM.\nНапример: 08:00",
		state.location.City, state.location.Timezone,
	))
}

func (h *Handler) handleTimeStep(c telebot.Context, userID int64, state *registrationState) error {
	timeText := strings.TrimSpace(c.Text())
	parsedTime, err := h.service.ParseNotifyTime(timeText)
	if err != nil {
		return c.Send("Не похоже на время в формате HH:MM. Попробуйте ещё раз, например 08:00:")
	}
	state.notifyTime = parsedTime

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := h.service.Register(ctx, model.ContactTelegram, strconv.FormatInt(userID, 10), state.location, state.notifyTime)
	if err != nil {
		return c.Send("Не получилось сохранить регистрацию, попробуйте позже.")
	}

	if state.cancelTimeout != nil {
		state.cancelTimeout()
	}
	h.states.delete(userID)

	return c.Send(fmt.Sprintf(
		"Готово! Буду присылать погоду в %s (%s) каждый день в %s.",
		user.City, user.Timezone, user.NotifyTime.Format("15:04"),
	))
}
