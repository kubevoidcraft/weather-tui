package weather

import "testing"

func TestWMOToText(t *testing.T) {
	tests := []struct {
		name string
		code int
		want string
	}{
		{"clear", 0, "Clear sky"},
		{"mainly clear", 1, "Mainly clear"},
		{"partly cloudy", 2, "Partly cloudy"},
		{"overcast", 3, "Overcast"},
		{"fog 45", 45, "Fog"},
		{"fog 48", 48, "Fog"},
		{"drizzle light", 51, "Drizzle"},
		{"drizzle heavy", 55, "Drizzle"},
		{"freezing drizzle", 56, "Freezing Drizzle"},
		{"rain", 63, "Rain"},
		{"freezing rain", 66, "Freezing Rain"},
		{"snow", 73, "Snow fall"},
		{"snow grains", 77, "Snow grains"},
		{"rain showers", 81, "Rain showers"},
		{"snow showers", 85, "Snow showers"},
		{"thunderstorm", 95, "Thunderstorm"},
		{"thunderstorm hail 96", 96, "Thunderstorm with hail"},
		{"thunderstorm hail 99", 99, "Thunderstorm with hail"},
		{"unknown negative", -1, "Unknown"},
		{"unknown high", 200, "Unknown"},
		{"unknown gap", 10, "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WMOToText(tt.code); got != tt.want {
				t.Errorf("WMOToText(%d) = %q, want %q", tt.code, got, tt.want)
			}
		})
	}
}
