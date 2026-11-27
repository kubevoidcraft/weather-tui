package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kubevoidcraft/weather-tui/internal/weather"
)

// redirectConfigHome points UserConfigDir at a temp directory so tests do not
// touch the caller's real configuration. It must be called as the first line
// of any test that calls Load, Save or GetConfigPath.
func redirectConfigHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	return dir
}

func TestGetConfigPath(t *testing.T) {
	redirectConfigHome(t)

	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(path) != "favorites.json" {
		t.Errorf("expected basename favorites.json, got %q", filepath.Base(path))
	}
	// Directory must exist.
	if info, err := os.Stat(filepath.Dir(path)); err != nil || !info.IsDir() {
		t.Errorf("expected config directory to exist, stat err=%v", err)
	}
}

func TestLoadWithoutExistingFileReturnsEmptyConfig(t *testing.T) {
	redirectConfigHome(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatalf("expected non-nil config")
	}
	if len(cfg.Favorites) != 0 {
		t.Errorf("expected no favorites, got %d", len(cfg.Favorites))
	}
	// First run must still carry unit defaults; an empty unit would be sent
	// verbatim as temperature_unit= / wind_speed_unit= to Open-Meteo.
	if cfg.TemperatureUnit != "celsius" || cfg.WindUnit != "kmh" {
		t.Errorf("expected default units on first run, got %q / %q", cfg.TemperatureUnit, cfg.WindUnit)
	}
}

func TestLoadAppliesUnitDefaults(t *testing.T) {
	redirectConfigHome(t)

	// Write a minimal config missing unit settings.
	path, _ := GetConfigPath()
	if err := os.WriteFile(path, []byte(`{"favorites":[]}`), 0600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TemperatureUnit != "celsius" {
		t.Errorf("expected default celsius, got %q", cfg.TemperatureUnit)
	}
	if cfg.WindUnit != "kmh" {
		t.Errorf("expected default kmh, got %q", cfg.WindUnit)
	}
}

func TestLoadPreservesExplicitUnits(t *testing.T) {
	redirectConfigHome(t)

	path, _ := GetConfigPath()
	payload := `{"favorites":[],"temperature_unit":"fahrenheit","wind_unit":"mph"}`
	if err := os.WriteFile(path, []byte(payload), 0600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TemperatureUnit != "fahrenheit" || cfg.WindUnit != "mph" {
		t.Errorf("unit preferences not preserved: %+v", cfg)
	}
}

func TestLoadRejectsMalformedJSON(t *testing.T) {
	redirectConfigHome(t)

	path, _ := GetConfigPath()
	if err := os.WriteFile(path, []byte("{not json"), 0600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := Load()
	if !errors.Is(err, ErrMalformedConfig) {
		t.Errorf("expected ErrMalformedConfig, got %v", err)
	}
}

func TestLoadOrRecoverMovesMalformedFileAside(t *testing.T) {
	redirectConfigHome(t)

	path, _ := GetConfigPath()
	corrupt := []byte(`{"favorites":[{"name":"Paris"`)
	if err := os.WriteFile(path, corrupt, 0600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cfg, err := LoadOrRecover()
	if !errors.Is(err, ErrMalformedConfig) {
		t.Fatalf("expected ErrMalformedConfig, got %v", err)
	}
	if cfg == nil || cfg.TemperatureUnit != "celsius" || cfg.WindUnit != "kmh" {
		t.Fatalf("expected default config, got %+v", cfg)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Errorf("expected corrupt file to be moved away from %s, stat err=%v", path, statErr)
	}
	backup := path + ".corrupt"
	saved, readErr := os.ReadFile(backup) //nolint:gosec // path is inside the test's temp dir.
	if readErr != nil || string(saved) != string(corrupt) {
		t.Errorf("expected original bytes preserved in .corrupt, got %q (err=%v)", saved, readErr)
	}

	// The next Save must not touch the preserved copy.
	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	if again, _ := os.ReadFile(backup); string(again) != string(corrupt) { //nolint:gosec // test temp dir.
		t.Errorf(".corrupt changed after Save: %q", again)
	}
}

func TestLoadOrRecoverPassesThroughValidAndMissing(t *testing.T) {
	redirectConfigHome(t)

	cfg, err := LoadOrRecover()
	if err != nil || cfg.TemperatureUnit != "celsius" {
		t.Fatalf("missing file: cfg=%+v err=%v", cfg, err)
	}

	path, _ := GetConfigPath()
	if err := os.WriteFile(path, []byte(`{"favorites":[],"wind_unit":"mph"}`), 0600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	cfg, err = LoadOrRecover()
	if err != nil || cfg.WindUnit != "mph" {
		t.Fatalf("valid file: cfg=%+v err=%v", cfg, err)
	}
	if _, statErr := os.Stat(path + ".corrupt"); !os.IsNotExist(statErr) {
		t.Errorf("valid file must not be moved aside")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	redirectConfigHome(t)

	cfg := &Config{
		Favorites: []weather.City{
			{Name: "Paris", Country: "FR", Lat: 48.85, Lon: 2.35, Timezone: "Europe/Paris"},
		},
		TemperatureUnit: "fahrenheit",
		WindUnit:        "mph",
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(loaded.Favorites) != 1 || loaded.Favorites[0].Name != "Paris" {
		t.Errorf("favorites not persisted: %+v", loaded.Favorites)
	}
	if loaded.TemperatureUnit != "fahrenheit" || loaded.WindUnit != "mph" {
		t.Errorf("units not persisted: %+v", loaded)
	}
}

func TestAddFavoriteAppends(t *testing.T) {
	cfg := &Config{}
	cfg.AddFavorite(weather.City{Name: "Berlin", Country: "DE"})
	if len(cfg.Favorites) != 1 || cfg.Favorites[0].Name != "Berlin" {
		t.Errorf("expected Berlin added, got %+v", cfg.Favorites)
	}
}

func TestAddFavoriteDeduplicatesByNameAndCountry(t *testing.T) {
	cfg := &Config{
		Favorites: []weather.City{
			{Name: "Paris", Country: "FR", Lat: 0, Lon: 0},
		},
	}
	// Adding the same city with updated coords should replace, not append.
	cfg.AddFavorite(weather.City{Name: "Paris", Country: "FR", Lat: 48.85, Lon: 2.35})
	if len(cfg.Favorites) != 1 {
		t.Errorf("expected 1 entry after dedupe, got %d", len(cfg.Favorites))
	}
	if cfg.Favorites[0].Lat != 48.85 {
		t.Errorf("expected coords to be updated, got %+v", cfg.Favorites[0])
	}
}

func TestAddFavoriteTreatsDifferentCountriesAsDistinct(t *testing.T) {
	cfg := &Config{
		Favorites: []weather.City{
			{Name: "Paris", Country: "FR"},
		},
	}
	cfg.AddFavorite(weather.City{Name: "Paris", Country: "US"})
	if len(cfg.Favorites) != 2 {
		t.Errorf("expected 2 entries for different-country cities, got %d", len(cfg.Favorites))
	}
}

func TestAddFavoriteCapsAtFive(t *testing.T) {
	cfg := &Config{}
	cities := []string{"A", "B", "C", "D", "E", "F"}
	for _, name := range cities {
		cfg.AddFavorite(weather.City{Name: name})
	}
	if len(cfg.Favorites) != 5 {
		t.Fatalf("expected 5 favorites after adding 6, got %d", len(cfg.Favorites))
	}
	// The earliest ("A") should have been evicted.
	if cfg.Favorites[0].Name != "B" {
		t.Errorf("expected first favorite to be B, got %q", cfg.Favorites[0].Name)
	}
}

func TestRemoveFavorite(t *testing.T) {
	cfg := &Config{
		Favorites: []weather.City{
			{Name: "A"}, {Name: "B"}, {Name: "C"},
		},
	}
	if !cfg.RemoveFavorite(weather.City{Name: "B"}) {
		t.Errorf("expected true for existing favorite")
	}
	if len(cfg.Favorites) != 2 || cfg.Favorites[1].Name != "C" {
		t.Errorf("unexpected favorites after removal: %+v", cfg.Favorites)
	}
	if cfg.RemoveFavorite(weather.City{Name: "Z"}) {
		t.Errorf("expected false for non-existent favorite")
	}
}

func TestRemoveFavoriteMatchesCountry(t *testing.T) {
	cfg := &Config{
		Favorites: []weather.City{
			{Name: "Paris", Country: "FR"}, {Name: "Paris", Country: "US"},
		},
	}
	if !cfg.RemoveFavorite(weather.City{Name: "Paris", Country: "US"}) {
		t.Fatalf("expected Paris (US) to be removed")
	}
	if len(cfg.Favorites) != 1 || cfg.Favorites[0].Country != "FR" {
		t.Errorf("wrong Paris removed: %+v", cfg.Favorites)
	}
}

func TestIsFavorite(t *testing.T) {
	cfg := &Config{Favorites: []weather.City{{Name: "Paris", Country: "FR"}}}
	if !cfg.IsFavorite(weather.City{Name: "Paris", Country: "FR"}) {
		t.Errorf("expected Paris (FR) to be a favorite")
	}
	if cfg.IsFavorite(weather.City{Name: "Paris", Country: "US"}) {
		t.Errorf("Paris (US) must not match Paris (FR)")
	}
}

// Sanity: ensure GetConfigPath fails gracefully when HOME is missing entirely
// (UserConfigDir errors). Only runs on Unix where removing HOME matters.
func TestGetConfigPathErrorsWithoutHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows uses AppData, not HOME")
	}
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	if _, err := GetConfigPath(); err == nil {
		t.Errorf("expected error when HOME is not set")
	}
}
