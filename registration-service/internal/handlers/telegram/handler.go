package telegram

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"gopkg.in/telebot.v4"
)

// UserStore — интерфейс поверх будущего слоя БД.
// Пока можно подставить in-memory реализацию для разработки бота
// без готовой Postgres/репозитория.
type UserStore interface {
	Save(u RegisteredUser) error
}

type RegisteredUser struct {
	TelegramID int64
	City       string
	Timezone   string // IANA, напр. "Europe/Moscow" — резолвится через Geocoder, не вводится вручную
	NotifyTime string // "HH:MM", парсинг в time.Time — на уровне сервиса/БД, не тут
}

// registrationStep — на каком шаге диалога находится пользователь.
type registrationStep int

const (
	stepAwaitingCity registrationStep = iota
	stepAwaitingTime
)

// registrationState — прогресс диалога one-message-at-a-time.
type registrationState struct {
	step       registrationStep
	city       string // нормализованное имя от geocoder
	timezone   string
	notifyTime string
}

// stateStore — потокобезопасное хранилище состояний регистрации.
// Апдейты от telebot могут прилетать конкурентно для разных пользователей,
// поэтому доступ к карте всегда идёт через мьютекс.
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

type Handler struct {
	bot      *telebot.Bot
	store    UserStore
	geocoder Geocoder
	states   *stateStore
}

func New(token string, store UserStore, geocoder Geocoder) (*Handler, error) {
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
		bot:      bot,
		store:    store,
		geocoder: geocoder,
		states:   newStateStore(),
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
	h.states.set(userID, &registrationState{step: stepAwaitingCity})
	return c.Send("Введите ваш город:")
}

func (h *Handler) cancel(c telebot.Context) error {
	if c.Sender() == nil {
		return nil
	}
	userID := c.Sender().ID
	if _, exists := h.states.get(userID); !exists {
		return c.Send("Сейчас нет активной регистрации.")
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
		// пользователь не в процессе регистрации — просто игнорируем текст
		return nil
	}

	switch state.step {
	case stepAwaitingCity:
		return h.handleCityStep(c, userID, state)
	case stepAwaitingTime:
		return h.handleTimeStep(c, userID, state)
	default:
		// не должно случиться, но подчищаем битое состояние вместо зависания
		h.states.delete(userID)
		return c.Send("Что-то пошло не так, начните заново: /register")
	}
}

func (h *Handler) handleCityStep(c telebot.Context, userID int64, state *registrationState) error {
	cityInput := strings.TrimSpace(c.Text())
	if cityInput == "" {
		return c.Send("Город не может быть пустым. Попробуйте ещё раз:")
	}

	// Таймаут на сетевой вызов, чтобы бот не подвисал, если geocoding API недоступен.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	loc, err := h.geocoder.Resolve(ctx, cityInput)
	switch {
	case errors.Is(err, ErrCityNotFound):
		return c.Send("Не смог найти такой город. Проверьте написание и попробуйте ещё раз:")
	case err != nil:
		// сервис геокодинга недоступен — не роняем регистрацию, даём пользователю повторить попытку
		return c.Send("Не получилось определить часовой пояс для этого города (сервис временно недоступен). Попробуйте ещё раз чуть позже:")
	}

	state.city = loc.City
	state.timezone = loc.Timezone
	state.step = stepAwaitingTime
	h.states.set(userID, state)

	return c.Send(fmt.Sprintf(
		"Город сохранён: %s (часовой пояс: %s)\n\nТеперь введите время рассылки в формате HH:MM.\nНапример: 08:00",
		state.city, state.timezone,
	))
}

func (h *Handler) handleTimeStep(c telebot.Context, userID int64, state *registrationState) error {
	timeText := strings.TrimSpace(c.Text())
	if _, err := time.Parse("15:04", timeText); err != nil {
		return c.Send("Не похоже на время в формате HH:MM. Попробуйте ещё раз, например 08:00:")
	}
	state.notifyTime = timeText

	user := RegisteredUser{
		TelegramID: userID,
		City:       state.city,
		Timezone:   state.timezone,
		NotifyTime: state.notifyTime,
	}
	if err := h.store.Save(user); err != nil {
		return c.Send("Не получилось сохранить регистрацию, попробуйте позже.")
	}

	h.states.delete(userID)

	return c.Send(fmt.Sprintf(
		"Готово! Буду присылать погоду в %s (%s) каждый день в %s.",
		state.city, state.timezone, state.notifyTime,
	))
}
