package weather

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCityLabel(t *testing.T) {
	tests := []struct {
		name string
		city City
		want string
	}{
		{
			name: "with country",
			city: City{Name: "Berlin", Country: "DE"},
			want: "Berlin (DE)",
		},
		{
			name: "without country",
			city: City{Name: "Atlantis"},
			want: "Atlantis",
		},
		{
			name: "empty country string",
			city: City{Name: "Solo", Country: ""},
			want: "Solo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.city.Label(); got != tt.want {
				t.Errorf("Label() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCityLocalTimeValidTimezone(t *testing.T) {
	c := City{Name: "Tokyo", Timezone: "Asia/Tokyo"}
	now := c.LocalTime()
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Skipf("tz database missing: %v", err)
	}
	if now.Location().String() != loc.String() {
		t.Errorf("expected location %q, got %q", loc.String(), now.Location().String())
	}
}

func TestCityLocalTimeEmptyTimezoneFallsBackToLocal(t *testing.T) {
	c := City{Name: "Nowhere"}
	now := c.LocalTime()
	// When timezone is empty, the function falls back to time.Now() which
	// uses the system local zone. We just verify we got a non-zero time.
	if now.IsZero() {
		t.Errorf("expected non-zero local time, got zero")
	}
}

func TestCityLocalTimeInvalidTimezoneFallsBackToLocal(t *testing.T) {
	c := City{Name: "Bogus", Timezone: "Not/AReal/Zone"}
	got := c.LocalTime()
	// Fallback path returns time.Now() in local zone; rough sanity check:
	// the call should not panic and the returned time should be close to now.
	delta := time.Since(got)
	if delta < -time.Second || delta > time.Second {
		t.Errorf("expected fallback time close to now, delta=%v", delta)
	}
}

// TestCityJSONRoundTrip ensures the City struct can be serialised and
// deserialised without loss. This protects Config persistence.
func TestCityJSONRoundTrip(t *testing.T) {
	orig := City{Name: "Paris", Country: "FR", Lat: 48.85, Lon: 2.35, Timezone: "Europe/Paris"}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var back City
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if back != orig {
		t.Errorf("round-trip mismatch: got %+v, want %+v", back, orig)
	}
}
