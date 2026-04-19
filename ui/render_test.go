package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/kubevoidcraft/weather-tui/internal/config"
	"github.com/kubevoidcraft/weather-tui/internal/weather"
	"github.com/rivo/tview"
)

func TestHourOfDay(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"2026-04-19T13:00", "13"},
		{"2026-01-01T00:00", "00"},
		{"2026-12-31T23:45", "23"},
		{"short", "??"},
		{"", "??"},
	}
	for _, tt := range tests {
		if got := hourOfDay(tt.raw); got != tt.want {
			t.Errorf("hourOfDay(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestSpaceOut3(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"a", "a"},
		{"ab", "a  b"},
		{"abc", "a  b  c"},
	}
	for _, tt := range tests {
		if got := spaceOut3(tt.in); got != tt.want {
			t.Errorf("spaceOut3(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestPadVisibleRightPads(t *testing.T) {
	got := padVisible("hi", 5)
	if got != "hi   " {
		t.Errorf("padVisible = %q, want %q", got, "hi   ")
	}
}

func TestPadVisibleIgnoresStyleTags(t *testing.T) {
	// The visible width of "[red]hi[-]" is 2, so padding to 5 should add 3 spaces.
	got := padVisible("[red]hi[-]", 5)
	if tview.TaggedStringWidth(got) != 5 {
		t.Errorf("expected visible width 5, got %d (%q)", tview.TaggedStringWidth(got), got)
	}
}

func TestPadVisibleDoesNotShrink(t *testing.T) {
	got := padVisible("abcdef", 3)
	if got != "abcdef" {
		t.Errorf("padVisible should not truncate, got %q", got)
	}
}

func TestSummarize(t *testing.T) {
	out := summarize([]float64{5, 10, 22, 17, 8}, "°C", "%.0f")
	// Expect "first -> max -> last °C" = "5 -> 22 -> 8 °C" wrapped in tview style tags.
	if !strings.Contains(out, "5") || !strings.Contains(out, "22") || !strings.Contains(out, "8") {
		t.Errorf("expected first/max/last in output, got %q", out)
	}
	if !strings.Contains(out, "°C") {
		t.Errorf("expected unit label in output, got %q", out)
	}
}

func TestSummarizeEmpty(t *testing.T) {
	if got := summarize(nil, "°C", "%.0f"); got != "" {
		t.Errorf("expected empty string for empty values, got %q", got)
	}
}

func TestSummarizePercent(t *testing.T) {
	out := summarize([]float64{0, 50, 100}, "%%", "%.0f")
	if !strings.Contains(out, "%") {
		t.Errorf("expected literal %% in output, got %q", out)
	}
}

func TestDailyWindIndexBounds(t *testing.T) {
	winds := []float64{10, 20, 30}
	if got := dailyWind(winds, 1); got != 20 {
		t.Errorf("expected 20, got %v", got)
	}
	if got := dailyWind(winds, 5); got != -1 {
		t.Errorf("expected -1 for out-of-range, got %v", got)
	}
	if got := dailyWind(winds, -1); got != -1 {
		t.Errorf("expected -1 for negative index, got %v", got)
	}
	if got := dailyWind(nil, 0); got != -1 {
		t.Errorf("expected -1 for nil slice, got %v", got)
	}
}

func TestFavoriteIndex(t *testing.T) {
	app := &App{Config: &config.Config{Favorites: []weather.City{
		{Name: "Berlin", Lat: 52.52, Lon: 13.41},
		{Name: "Paris", Lat: 48.85, Lon: 2.35},
	}}}

	if got := app.favoriteIndex(weather.City{Name: "Paris", Lat: 48.85, Lon: 2.35}); got != 1 {
		t.Errorf("expected index 1, got %d", got)
	}
	if got := app.favoriteIndex(weather.City{Name: "Tokyo", Lat: 35.68, Lon: 139.69}); got != -1 {
		t.Errorf("expected -1 for unknown city, got %d", got)
	}
	// Name collision but different coords should NOT match.
	if got := app.favoriteIndex(weather.City{Name: "Paris", Lat: 33.66, Lon: -95.55}); got != -1 {
		t.Errorf("expected -1 when coords differ, got %d", got)
	}
}

func TestUnitLabels(t *testing.T) {
	cases := []struct {
		temp, wind     string
		wantT, wantW   string
		describe       string
	}{
		{"celsius", "kmh", "°C", "km/h", "defaults"},
		{"fahrenheit", "ms", "°F", "m/s", "imperial temp + m/s wind"},
		{"celsius", "mph", "°C", "mph", "metric temp + mph wind"},
		{"", "", "°C", "km/h", "empty values use celsius/kmh baseline"},
	}
	for _, tt := range cases {
		t.Run(tt.describe, func(t *testing.T) {
			app := &App{Config: &config.Config{TemperatureUnit: tt.temp, WindUnit: tt.wind}}
			gotT, gotW := app.unitLabels()
			if gotT != tt.wantT || gotW != tt.wantW {
				t.Errorf("unitLabels() = %q,%q want %q,%q", gotT, gotW, tt.wantT, tt.wantW)
			}
		})
	}
}

func TestHourlyStartIndexSkipsPast(t *testing.T) {
	// City in a fixed timezone so the test is reproducible.
	c := weather.City{Name: "Berlin", Timezone: "Europe/Berlin"}
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Skipf("tz data missing: %v", err)
	}

	// Build a sequence centred on "now": 3 past hours, then 3 future hours.
	base := time.Now().In(loc).Truncate(time.Hour)
	var times []string
	for offset := -3; offset <= 3; offset++ {
		times = append(times, base.Add(time.Duration(offset)*time.Hour).Format("2006-01-02T15:04"))
	}

	idx := hourlyStartIndex(times, c)
	if idx != 3 {
		t.Errorf("expected start index 3 (current hour), got %d", idx)
	}
}

func TestHourlyStartIndexEmpty(t *testing.T) {
	c := weather.City{Timezone: "UTC"}
	if got := hourlyStartIndex(nil, c); got != -1 {
		t.Errorf("expected -1 for empty slice, got %d", got)
	}
}

func TestHourlyStartIndexAllPast(t *testing.T) {
	c := weather.City{Timezone: "UTC"}
	// Timestamps from a clearly past date.
	times := []string{"2000-01-01T00:00", "2000-01-01T01:00"}
	if got := hourlyStartIndex(times, c); got != -1 {
		t.Errorf("expected -1 when all timestamps are past, got %d", got)
	}
}

func TestRenderCurrentWeather(t *testing.T) {
	fc := &weather.Forecast{
		Current: weather.CurrentWeather{
			Temperature2m: 15.3,
			WindSpeed10m:  12.4,
			WeatherCode:   2,
		},
	}
	lines := renderCurrentWeather(fc, "°C", "km/h")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d: %#v", len(lines), lines)
	}
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"Current Weather", "15.3", "12.4", "Partly cloudy"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, joined)
		}
	}
}

func TestRenderHourlyPanelFormatting(t *testing.T) {
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		t.Fatalf("cannot load UTC: %v", err)
	}

	// Build a forecast with hours centred on now so hourlyStartIndex finds a
	// window to render.
	base := time.Now().In(loc).Truncate(time.Hour)
	var hourly weather.HourlyWeather
	for i := -2; i < 14; i++ {
		hourly.Time = append(hourly.Time, base.Add(time.Duration(i)*time.Hour).Format("2006-01-02T15:04"))
		hourly.Temperature2m = append(hourly.Temperature2m, 10+float64(i))
		hourly.WindSpeed10m = append(hourly.WindSpeed10m, 5+float64(i)/2)
		hourly.PrecipitationProbability = append(hourly.PrecipitationProbability, float64(i)*5)
	}

	fc := &weather.Forecast{Hourly: hourly}
	city := weather.City{Timezone: "UTC"}
	lines := renderHourlyPanel(fc, city, "°C", "km/h")
	if len(lines) == 0 {
		t.Fatalf("expected non-empty hourly panel")
	}
	header := lines[0]
	if !strings.Contains(header, "Next") || !strings.Contains(header, "hours") {
		t.Errorf("expected header row, got %q", header)
	}
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"Time", "🌡", "💨", "🌧", "°C", "km/h"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected hourly panel to contain %q, got:\n%s", want, joined)
		}
	}
}

func TestRenderHourlyPanelReturnsNilWhenNoWindow(t *testing.T) {
	fc := &weather.Forecast{Hourly: weather.HourlyWeather{
		Time: []string{"2000-01-01T00:00"}, // Entirely in the past.
	}}
	if lines := renderHourlyPanel(fc, weather.City{Timezone: "UTC"}, "°C", "km/h"); lines != nil {
		t.Errorf("expected nil for past-only data, got %d lines", len(lines))
	}
}

func TestRenderDailyForecastIncludesHeaderAndWind(t *testing.T) {
	fc := &weather.Forecast{Daily: weather.DailyWeather{
		Time:             []string{"2026-04-19", "2026-04-20"},
		WeatherCode:      []int{0, 63},
		Temperature2mMax: []float64{22.5, 18.1},
		Temperature2mMin: []float64{12.3, 10.0},
		WindSpeed10mMax:  []float64{15.2, 28.7},
	}}
	out := renderDailyForecast(fc, "°C", "km/h")
	// Header must be present with column names.
	for _, want := range []string{"7-Day Forecast", "Date", "Condition", "Min", "Max", "Wind"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected header token %q, got:\n%s", want, out)
		}
	}
	// Data rows must include dates, min/max temps with arrows, wind values, condition text.
	for _, want := range []string{"2026-04-19", "2026-04-20", "▼ 12.3", "▲ 22.5", "Clear sky", "Rain", "💨 15.2", "28.7"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected row token %q, got:\n%s", want, out)
		}
	}
}

func TestRenderDailyForecastSkipsMissingWind(t *testing.T) {
	fc := &weather.Forecast{Daily: weather.DailyWeather{
		Time:             []string{"2026-04-19"},
		WeatherCode:      []int{0},
		Temperature2mMax: []float64{22.5},
		Temperature2mMin: []float64{12.3},
		// No WindSpeed10mMax - simulates older API responses.
	}}
	out := renderDailyForecast(fc, "°C", "km/h")
	if strings.Contains(out, "💨") {
		t.Errorf("expected no wind column when data missing, got:\n%s", out)
	}
}
