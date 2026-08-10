package geocoding

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type OpenMeteoGeocoder struct {
	httpClient *http.Client
}

func NewOpenMeteoGeocoder() *OpenMeteoGeocoder {
	return &OpenMeteoGeocoder{
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

type openMeteoResponse struct {
	Results []struct {
		Name      string  `json:"name"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Timezone  string  `json:"timezone"`
		Country   string  `json:"country"`
	} `json:"results"`
}

func (g *OpenMeteoGeocoder) Resolve(ctx context.Context, city string) (GeoLocation, error) {
	endpoint := "https://geocoding-api.open-meteo.com/v1/search?" + url.Values{
		"name":     {city},
		"count":    {"1"},
		"language": {"ru"},
		"format":   {"json"},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return GeoLocation{}, fmt.Errorf("build geocoding request: %w", err)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return GeoLocation{}, fmt.Errorf("call geocoding api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return GeoLocation{}, fmt.Errorf("geocoding api status %d", resp.StatusCode)
	}

	var parsed openMeteoResponse

	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return GeoLocation{}, fmt.Errorf("decode geocoding api response: %w", err)
	}

	if len(parsed.Results) == 0 {
		return GeoLocation{}, ErrCityNotFound
	}

	top := parsed.Results[0]
	if top.Timezone == "" {
		return GeoLocation{}, fmt.Errorf("time zone is missing")
	}

	return GeoLocation{
		City:      top.Name,
		Latitude:  top.Latitude,
		Longitude: top.Longitude,
		Timezone:  top.Timezone,
	}, nil
}
