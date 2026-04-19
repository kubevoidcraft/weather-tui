package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Package-level variables (not consts) so that tests can point them at an
// httptest server. Production code never mutates these at runtime.
var (
	geomapAPIURL  = "https://geocoding-api.open-meteo.com/v1/search"
	weatherAPIURL = "https://api.open-meteo.com/v1/forecast"
)

const (
	// clientTimeout bounds the entire request (connect + read). Chosen so a
	// spotty connection fails fast and the TUI can fall back to cached data
	// instead of hanging.
	clientTimeout = 10 * time.Second

	// maxResponseBytes caps the JSON payload we are willing to decode.
	// Open-Meteo responses are a few KiB; 1 MiB is a generous defensive
	// ceiling against a misbehaving or compromised upstream.
	maxResponseBytes = 1 << 20

	// userAgent identifies this client to Open-Meteo. The service's usage
	// guidelines ask third-party clients to set one so the operators can
	// distinguish apps when investigating traffic anomalies.
	userAgent = "weather-tui/1.x (+https://github.com/kubevoidcraft/weather-tui)"
)

// ErrQueryTooShort is returned when a search query is under the minimum
// length required by the geocoding API. Exported so callers can distinguish
// user-input issues from network failures.
var ErrQueryTooShort = errors.New("query too short (minimum 3 characters)")

// SearchResult represents the top-level JSON response from the Open-Meteo Geocoding API.
type SearchResult struct {
	Results []City `json:"results"`
}

// Forecast represents the top-level JSON response from the Open-Meteo Weather API.
type Forecast struct {
	Latitude  float64        `json:"latitude"`
	Longitude float64        `json:"longitude"`
	Timezone  string         `json:"timezone"`
	Current   CurrentWeather `json:"current"`
	Hourly    HourlyWeather  `json:"hourly"`
	Daily     DailyWeather   `json:"daily"`
}

// CurrentWeather parses the current weather data.
type CurrentWeather struct {
	Time          string  `json:"time"`
	Temperature2m float64 `json:"temperature_2m"`
	WindSpeed10m  float64 `json:"wind_speed_10m"`
	WeatherCode   int     `json:"weather_code"`
}

// HourlyWeather parses the hourly forecast arrays used to drive the 12-hour
// outlook panel. All slices share the same length and are index-aligned with
// Time. Open-Meteo returns values starting at 00:00 local time of the current
// day, so callers are expected to slice the relevant window themselves.
type HourlyWeather struct {
	Time                     []string  `json:"time"`
	Temperature2m            []float64 `json:"temperature_2m"`
	WindSpeed10m             []float64 `json:"wind_speed_10m"`
	PrecipitationProbability []float64 `json:"precipitation_probability"`
}

// DailyWeather parses the 7-day forecast data.
type DailyWeather struct {
	Time             []string  `json:"time"`
	WeatherCode      []int     `json:"weather_code"`
	Temperature2mMax []float64 `json:"temperature_2m_max"`
	Temperature2mMin []float64 `json:"temperature_2m_min"`
	WindSpeed10mMax  []float64 `json:"wind_speed_10m_max"`
}

// httpClient is reused across calls so the underlying connection pool is
// amortised. The timeout provides a hard upper bound even if a caller forgets
// to set a context deadline.
var httpClient = &http.Client{Timeout: clientTimeout}

// doJSON issues a GET request that honours ctx, enforces our User-Agent,
// caps the response body, and decodes the JSON into out. Extracted so
// SearchCities and GetForecast share identical network semantics.
func doJSON(ctx context.Context, reqURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("api returned status: %s", resp.Status)
	}

	// Bound the JSON we decode so a runaway upstream cannot exhaust memory.
	body := io.LimitReader(resp.Body, maxResponseBytes)
	if err := json.NewDecoder(body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// SearchCities calls the Open-Meteo Geocoding API to find cities matching the
// query. The provided context governs cancellation and per-call deadlines;
// callers should pass a context with a reasonable timeout (or rely on the
// client's built-in 10s ceiling).
func SearchCities(ctx context.Context, query string) ([]City, error) {
	if len(query) < 3 {
		return nil, ErrQueryTooShort
	}

	reqURL := fmt.Sprintf("%s?name=%s&count=10&language=en&format=json", geomapAPIURL, url.QueryEscape(query))

	var res SearchResult
	if err := doJSON(ctx, reqURL, &res); err != nil {
		return nil, fmt.Errorf("geocoding %w", err)
	}
	return res.Results, nil
}

// GetForecast calls the Open-Meteo Forecast API for the given coordinates,
// timezone, and units. See SearchCities for context semantics.
func GetForecast(ctx context.Context, lat, lon float64, timezone, tempUnit, windUnit string) (*Forecast, error) {
	// Open-Meteo requires timezone to align the daily aggregations correctly
	if timezone == "" {
		timezone = "auto"
	}

	reqURL := fmt.Sprintf(
		"%s?latitude=%f&longitude=%f&current=temperature_2m,wind_speed_10m,weather_code&hourly=temperature_2m,wind_speed_10m,precipitation_probability&daily=weather_code,temperature_2m_max,temperature_2m_min,wind_speed_10m_max&timezone=%s&temperature_unit=%s&wind_speed_unit=%s",
		weatherAPIURL, lat, lon, url.QueryEscape(timezone), url.QueryEscape(tempUnit), url.QueryEscape(windUnit),
	)

	var res Forecast
	if err := doJSON(ctx, reqURL, &res); err != nil {
		return nil, fmt.Errorf("weather %w", err)
	}
	return &res, nil
}
