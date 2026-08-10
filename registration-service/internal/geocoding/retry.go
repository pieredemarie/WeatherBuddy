package geocoding

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type retryingGeocoder struct {
	inner       Geocoder
	maxAttempts int
	baseDelay   time.Duration
}

func WithRetry(inner Geocoder, maxAttempts int, baseDelay time.Duration) Geocoder {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	return &retryingGeocoder{inner: inner, maxAttempts: maxAttempts, baseDelay: baseDelay}
}

func (g *retryingGeocoder) Resolve(ctx context.Context, city string) (GeoLocation, error) {
	var lastErr error

	for attempt := 0; attempt < g.maxAttempts; attempt++ {
		loc, err := g.inner.Resolve(ctx, city)
		if err == nil {
			return loc, nil
		}
		if errors.Is(err, ErrCityNotFound) {
			return GeoLocation{}, err
		}

		lastErr = err
		if attempt == g.maxAttempts-1 {
			break
		}

		delay := g.baseDelay * time.Duration(1<<attempt) // 1x, 2x, 4x, ...
		select {
		case <-ctx.Done():
			return GeoLocation{}, ctx.Err()
		case <-time.After(delay):
		}
	}

	return GeoLocation{}, fmt.Errorf("geocoding failed after %d attempts: %w", g.maxAttempts, lastErr)
}
