# SKILL.md
## Weather TUI Required Skills

- **TUI Framework**: Must be proficient in Go TUI applications. Since the application is inspired by `k9s`, `github.com/rivo/tview` is the recommended building block (k9s relies on it under the hood).
- **APIs (No-token)**:
  - **Weather Data**: Open-Meteo (`api.open-meteo.com/v1/forecast`) provides robust, rich functionality without requiring token registration.
  - **City Search (Geocoding)**: Open-Meteo Geocoding (`geocoding-api.open-meteo.com/v1/search`) works perfectly for the 3-character autocomplete functionality.
- **Go Context & Concurrency**: Vital for non-blocking UI during API fetching.
- **Time/Timezone Handling**: Required for offline mode and displaying timezone/time contextually.
- **Tools validation**: Ability to routinely run `golangci-lint run` before validating changes.
