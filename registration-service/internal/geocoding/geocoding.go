package geocoding

import (
	"context"
	"errors"
)

var ErrCityNotFound = errors.New("city not found")

type GeoLocation struct {
	City      string
	Latitude  float64
	Longitude float64
	Timezone  string // for example: "Europe/Moscow"
}

type Geocoder interface {
	Resolve(ctx context.Context, city string) (GeoLocation, error)
}
