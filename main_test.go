package main

import "testing"

func TestVersionString(t *testing.T) {
	cases := []struct {
		name, ver, commit, date, build, want string
	}{
		{"goreleaser build", "0.1.0", "abc123", "2026-11-25", "", "weather-tui 0.1.0 (commit abc123, built 2026-11-25)"},
		{"go install with tag", "dev", "", "", "v0.1.0", "weather-tui v0.1.0"},
		{"local go build", "dev", "", "", "(devel)", "weather-tui dev"},
		{"no build info", "dev", "", "", "", "weather-tui dev"},
		{"ldflags beat build info", "0.2.0", "", "", "v0.1.0", "weather-tui 0.2.0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := versionString(tc.ver, tc.commit, tc.date, tc.build); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
