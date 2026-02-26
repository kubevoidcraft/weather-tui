// Package weather provides models and API clients for weather data.
package weather

import "time"

// City represents a geographical location with its exact coordinates and timezone.
// This structure is essential for calculating offline local time and querying the Open-Meteo API.
type City struct {
	Name     string  `json:"name"`
	Country  string  `json:"country,omitempty"`
	Lat      float64 `json:"latitude"`
	Lon      float64 `json:"longitude"`
	Timezone string  `json:"timezone"` // e.g. "Europe/Berlin"
}

// LocalTime returns the current local time for the city based on its Timezone field.
// If the timezone is invalid or undefined, it falls back to the system's local time.
func (c City) LocalTime() time.Time {
	if c.Timezone == "" {
		return time.Now()
	}
	loc, err := time.LoadLocation(c.Timezone)
	if err != nil {
		// Fallback to local system time if timezone database is missing or string is invalid.
		return time.Now()
	}
	return time.Now().In(loc)
}

// Label returns a formatted string for UI representation, such as "London (GB)".
func (c City) Label() string {
	if c.Country != "" {
		return c.Name + " (" + c.Country + ")"
	}
	return c.Name
}
