// Package config handles reading and writing application state, such as favorite cities.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kubevoidcraft/weather-tui/internal/weather"
)

// Config represents the user's persistent application data.
type Config struct {
	Favorites       []weather.City `json:"favorites"`
	TemperatureUnit string         `json:"temperature_unit,omitempty"` // "celsius", "fahrenheit"
	WindUnit        string         `json:"wind_unit,omitempty"`        // "kmh", "ms", "mph"
}

// GetConfigPath returns the absolute path to the configuration file (e.g. ~/.config/weather-tui/favorites.json).
func GetConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not determine user config dir: %w", err)
	}
	appDir := filepath.Join(configDir, "weather-tui")
	// Ensure directory exists
	if err := os.MkdirAll(appDir, 0750); err != nil {
		return "", fmt.Errorf("could not create config directory: %w", err)
	}
	return filepath.Join(appDir, "favorites.json"), nil
}

// Load reads the application configuration from the disk.
// Returns an empty configuration (with no favorites) if the file does not exist.
func Load() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	//nolint:gosec // target path is controlled exclusively by UserConfigDir, not external input.
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Configuration doesn't exist yet (first run); return an empty
			// config that still carries the unit defaults, otherwise the
			// first forecast request goes out with empty unit parameters.
			cfg := &Config{Favorites: []weather.City{}}
			cfg.applyDefaults()
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	cfg.applyDefaults()

	return &cfg, nil
}

// applyDefaults fills in unit preferences that are missing from c.
func (c *Config) applyDefaults() {
	if c.TemperatureUnit == "" {
		c.TemperatureUnit = "celsius"
	}
	if c.WindUnit == "" {
		c.WindUnit = "kmh"
	}
}

// Save writes the current configuration back to the disk.
func (c *Config) Save() error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// sameFavorite reports whether a and b refer to the same favorite. Name and
// country together are the identity used by every favorites operation, so
// "Paris (FR)" and "Paris (US)" are distinct entries.
func sameFavorite(a, b weather.City) bool {
	return a.Name == b.Name && a.Country == b.Country
}

// IsFavorite reports whether city is in the favorites list.
func (c *Config) IsFavorite(city weather.City) bool {
	for _, fav := range c.Favorites {
		if sameFavorite(fav, city) {
			return true
		}
	}
	return false
}

// AddFavorite adds a new city to the favorites list. Keeps max 5 items. Returns true if added/updated.
func (c *Config) AddFavorite(city weather.City) bool {
	// Avoid duplicates
	for i, fav := range c.Favorites {
		if sameFavorite(fav, city) {
			// Update timezone/coords just in case
			c.Favorites[i] = city
			return true
		}
	}

	c.Favorites = append(c.Favorites, city)

	// Keep max 5
	if len(c.Favorites) > 5 {
		// Remove earliest
		c.Favorites = c.Favorites[1:]
	}
	return true
}

// RemoveFavorite removes city from the favorites list, matching on name and
// country. Returns true if removed.
func (c *Config) RemoveFavorite(city weather.City) bool {
	for i, f := range c.Favorites {
		if sameFavorite(f, city) {
			c.Favorites = append(c.Favorites[:i], c.Favorites[i+1:]...)
			return true
		}
	}
	return false
}
