// Package config handles reading and writing application state, such as favorite cities.
package config

import (
	"encoding/json"
	"errors"
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

// ErrMalformedConfig is returned by Load when the config file exists but is
// not valid JSON for Config.
var ErrMalformedConfig = errors.New("config file is malformed")

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
		return nil, fmt.Errorf("%w: %w", ErrMalformedConfig, err)
	}
	cfg.applyDefaults()

	return &cfg, nil
}

// LoadOrRecover is Load for application startup: it always returns a usable
// config with unit defaults applied. If the file exists but is malformed, it
// is renamed to "<path>.corrupt" (replacing an older one) so that the next
// Save cannot silently destroy the user's favorites, and the returned error
// says where it went. Callers should show the error and keep running.
func LoadOrRecover() (*Config, error) {
	cfg, err := Load()
	if err == nil {
		return cfg, nil
	}

	fallback := &Config{Favorites: []weather.City{}}
	fallback.applyDefaults()
	if !errors.Is(err, ErrMalformedConfig) {
		return fallback, err
	}

	path, pathErr := GetConfigPath()
	if pathErr != nil {
		return fallback, err
	}
	backup := path + ".corrupt"
	if renameErr := os.Rename(path, backup); renameErr != nil {
		return fallback, fmt.Errorf("%w (could not move it aside: %w)", err, renameErr)
	}
	return fallback, fmt.Errorf("%w; moved it to %s and started with defaults", err, backup)
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

	if err := writeFileAtomic(path, data); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// writeFileAtomic writes data to a temporary file next to path and renames
// it over path. Rename is atomic on the same filesystem, so a crash or full
// disk mid-write leaves either the old file or the new one, never a
// truncated one. os.CreateTemp creates the file with 0600 permissions.
func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// After a successful rename tmpName no longer exists and Remove is a
	// harmless no-op; on any failure it cleans up the partial file.
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
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
