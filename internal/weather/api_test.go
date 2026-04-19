package weather

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// withStubbedURL temporarily replaces a package-level URL variable for the
// duration of a test and restores it afterwards.
func withStubbedURL(target *string, replacement string, fn func()) {
	original := *target
	defer func() { *target = original }()
	*target = replacement
	fn()
}

func TestSearchCitiesRejectsShortQuery(t *testing.T) {
	_, err := SearchCities(context.Background(), "ab")
	if err == nil {
		t.Fatalf("expected error for query shorter than 3 chars")
	}
	if !errors.Is(err, ErrQueryTooShort) {
		t.Errorf("expected ErrQueryTooShort, got %v", err)
	}
}

func TestSearchCitiesParsesResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "name=paris") {
			t.Errorf("expected name=paris in query, got %q", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"results":[
			{"name":"Paris","country":"FR","latitude":48.85,"longitude":2.35,"timezone":"Europe/Paris"},
			{"name":"Paris","country":"US","latitude":33.66,"longitude":-95.55,"timezone":"America/Chicago"}
		]}`))
	}))
	defer srv.Close()

	withStubbedURL(&geomapAPIURL, srv.URL, func() {
		cities, err := SearchCities(context.Background(), "paris")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cities) != 2 {
			t.Fatalf("expected 2 results, got %d", len(cities))
		}
		if cities[0].Name != "Paris" || cities[0].Country != "FR" {
			t.Errorf("unexpected first result: %+v", cities[0])
		}
	})
}

func TestSearchCitiesSurfacesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	withStubbedURL(&geomapAPIURL, srv.URL, func() {
		if _, err := SearchCities(context.Background(), "paris"); err == nil {
			t.Errorf("expected error on non-200 response")
		}
	})
}

func TestSearchCitiesRejectsMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results": "not-an-array"`))
	}))
	defer srv.Close()

	withStubbedURL(&geomapAPIURL, srv.URL, func() {
		if _, err := SearchCities(context.Background(), "paris"); err == nil {
			t.Errorf("expected error on malformed json")
		}
	})
}

func TestSearchCitiesSendsUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer srv.Close()

	withStubbedURL(&geomapAPIURL, srv.URL, func() {
		if _, err := SearchCities(context.Background(), "paris"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(gotUA, "weather-tui") {
		t.Errorf("expected User-Agent to identify weather-tui, got %q", gotUA)
	}
}

func TestSearchCitiesRespectsContextCancellation(t *testing.T) {
	// Handler deliberately blocks until the client disconnects so we can
	// assert that cancelling the context aborts the request promptly.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	withStubbedURL(&geomapAPIURL, srv.URL, func() {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()
		start := time.Now()
		_, err := SearchCities(ctx, "paris")
		if err == nil {
			t.Fatalf("expected cancellation error")
		}
		if time.Since(start) > time.Second {
			t.Errorf("cancellation did not interrupt request promptly (took %s)", time.Since(start))
		}
	})
}

func TestGetForecastRequestsExpectedFields(t *testing.T) {
	var capturedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{
			"latitude":48.85,
			"longitude":2.35,
			"timezone":"Europe/Paris",
			"current":{"time":"2026-04-19T12:00","temperature_2m":14.2,"wind_speed_10m":5.1,"weather_code":2},
			"hourly":{
				"time":["2026-04-19T00:00","2026-04-19T01:00"],
				"temperature_2m":[10,11],
				"wind_speed_10m":[3,4],
				"precipitation_probability":[0,5]
			},
			"daily":{
				"time":["2026-04-19"],
				"weather_code":[2],
				"temperature_2m_max":[18.5],
				"temperature_2m_min":[9.3],
				"wind_speed_10m_max":[20.1]
			}
		}`))
	}))
	defer srv.Close()

	withStubbedURL(&weatherAPIURL, srv.URL, func() {
		fc, err := GetForecast(context.Background(), 48.85, 2.35, "Europe/Paris", "celsius", "kmh")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Validate the request query string had the parameters we care about.
		wantSubstrings := []string{
			"latitude=48.85",
			"longitude=2.35",
			"temperature_2m",
			"wind_speed_10m",
			"precipitation_probability",
			"wind_speed_10m_max",
			"temperature_unit=celsius",
			"wind_speed_unit=kmh",
		}
		for _, s := range wantSubstrings {
			if !strings.Contains(capturedQuery, s) {
				t.Errorf("expected query to contain %q, got %q", s, capturedQuery)
			}
		}

		// Spot-check parsed response.
		if fc.Current.Temperature2m != 14.2 {
			t.Errorf("current temp mismatch: %v", fc.Current.Temperature2m)
		}
		if len(fc.Hourly.Temperature2m) != 2 || fc.Hourly.Temperature2m[1] != 11 {
			t.Errorf("unexpected hourly: %+v", fc.Hourly)
		}
		if len(fc.Daily.WindSpeed10mMax) != 1 || fc.Daily.WindSpeed10mMax[0] != 20.1 {
			t.Errorf("expected wind_speed_10m_max, got %+v", fc.Daily.WindSpeed10mMax)
		}
	})
}

func TestGetForecastDefaultsTimezoneToAuto(t *testing.T) {
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"latitude":0,"longitude":0,"timezone":"UTC","current":{"time":"","temperature_2m":0,"wind_speed_10m":0,"weather_code":0},"hourly":{"time":[]},"daily":{"time":[]}}`))
	}))
	defer srv.Close()

	withStubbedURL(&weatherAPIURL, srv.URL, func() {
		if _, err := GetForecast(context.Background(), 0, 0, "", "celsius", "kmh"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(captured, "timezone=auto") {
			t.Errorf("expected timezone=auto in query, got %q", captured)
		}
	})
}

func TestGetForecastSurfacesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	withStubbedURL(&weatherAPIURL, srv.URL, func() {
		if _, err := GetForecast(context.Background(), 0, 0, "UTC", "celsius", "kmh"); err == nil {
			t.Errorf("expected error for 500 response")
		}
	})
}

// TestGetForecastCapsResponseBody ensures an oversized payload is rejected at
// decode time instead of being fully consumed into memory. We emit a valid
// JSON prefix followed by padding that pushes total length past the 1 MiB cap;
// LimitReader truncates mid-document, which the decoder then reports.
func TestGetForecastCapsResponseBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Write a minimal valid-looking start, then flood with whitespace
		// past the body cap so the decoder runs out of input before the
		// closing brace.
		_, _ = w.Write([]byte(`{"latitude":0,"longitude":0,"timezone":"UTC","padding":"`))
		chunk := strings.Repeat("x", 64*1024)
		// 1 MiB cap / 64 KiB = 16 chunks; write 20 to be safely over.
		for range 20 {
			_, _ = w.Write([]byte(chunk))
		}
	}))
	defer srv.Close()

	withStubbedURL(&weatherAPIURL, srv.URL, func() {
		if _, err := GetForecast(context.Background(), 0, 0, "UTC", "celsius", "kmh"); err == nil {
			t.Errorf("expected error when response exceeds size cap")
		}
	})
}
