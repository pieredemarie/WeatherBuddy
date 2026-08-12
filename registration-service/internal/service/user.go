package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"registration-service/internal/geocoding"
	"registration-service/internal/model"
)

var ErrInvalidNotifyTime = errors.New("invalid notify time format, expected HH:MM")

const notifyTimeLayout = "15:04"

type UserStore interface {
	Save(ctx context.Context, u model.User) error
}

type RegistrationService struct {
	geocoder geocoding.Geocoder
	store    UserStore
}

func NewRegistrationService(geocoder geocoding.Geocoder, store UserStore) *RegistrationService {
	return &RegistrationService{geocoder: geocoder, store: store}
}

func (s *RegistrationService) ResolveCity(ctx context.Context, city string) (geocoding.GeoLocation, error) {
	return s.geocoder.Resolve(ctx, city)
}

func (s *RegistrationService) ParseNotifyTime(text string) (time.Time, error) {
	t, err := time.Parse(notifyTimeLayout, text)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %q", ErrInvalidNotifyTime, text)
	}
	return t, nil
}

func (s *RegistrationService) Register(
	ctx context.Context,
	contactType model.ContactType,
	contactValue string,
	loc geocoding.GeoLocation,
	notifyTime time.Time,
) (model.User, error) {
	user := model.User{
		ContactType:  contactType,
		ContactValue: contactValue,
		City:         loc.City,
		Latitude:     loc.Latitude,
		Longitude:    loc.Longitude,
		Timezone:     loc.Timezone,
		NotifyTime:   notifyTime,
	}

	if err := s.store.Save(ctx, user); err != nil {
		return model.User{}, err
	}
	return user, nil
}
