package weather

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	geomapAPIURL  = "https://geocoding-api.open-meteo.com/v1/search"
	weatherAPIURL = "https://api.open-meteo.com/v1/forecast"
	clientTimeout = 10 * time.Second
)

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
	Daily     DailyWeather   `json:"daily"`
}

// CurrentWeather parses the current weather data.
type CurrentWeather struct {
	Time          string  `json:"time"`
	Temperature2m float64 `json:"temperature_2m"`
	WindSpeed10m  float64 `json:"wind_speed_10m"`
	WeatherCode   int     `json:"weather_code"`
}

// DailyWeather parses the 7-day forecast data.
type DailyWeather struct {
	Time             []string  `json:"time"`
	WeatherCode      []int     `json:"weather_code"`
	Temperature2mMax []float64 `json:"temperature_2m_max"`
	Temperature2mMin []float64 `json:"temperature_2m_min"`
}

// SearchCities calls the Open-Meteo Geocoding API to find cities matching the query.
func SearchCities(query string) ([]City, error) {
	if len(query) < 3 {
		return nil, fmt.Errorf("query too short (minimum 3 characters)")
	}

	client := &http.Client{Timeout: clientTimeout}

	reqURL := fmt.Sprintf("%s?name=%s&count=10&language=en&format=json", geomapAPIURL, url.QueryEscape(query))
	resp, err := client.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("geocoding api request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geocoding api returned status: %s", resp.Status)
	}

	var res SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode geocoding response: %w", err)
	}

	return res.Results, nil
}

// GetForecast calls the Open-Meteo Forecast API for the given coordinates, timezone, and units.
func GetForecast(lat, lon float64, timezone, tempUnit, windUnit string) (*Forecast, error) {
	client := &http.Client{Timeout: clientTimeout}

	// Open-Meteo requires timezone to align the daily aggregations correctly
	if timezone == "" {
		timezone = "auto"
	}

	reqURL := fmt.Sprintf(
		"%s?latitude=%f&longitude=%f&current=temperature_2m,wind_speed_10m,weather_code&daily=weather_code,temperature_2m_max,temperature_2m_min&timezone=%s&temperature_unit=%s&wind_speed_unit=%s",
		weatherAPIURL, lat, lon, url.QueryEscape(timezone), url.QueryEscape(tempUnit), url.QueryEscape(windUnit),
	)

	resp, err := client.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("weather api request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather api returned status: %s", resp.Status)
	}

	var res Forecast
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode weather response: %w", err)
	}

	return &res, nil
}
