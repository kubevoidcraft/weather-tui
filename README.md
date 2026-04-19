# Weather TUI

[![CI](https://github.com/kubevoidcraft/weather-tui/actions/workflows/ci.yml/badge.svg)](https://github.com/kubevoidcraft/weather-tui/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubevoidcraft/weather-tui)](https://goreportcard.com/report/github.com/kubevoidcraft/weather-tui)

A terminal-based interface (TUI) for weather forecasts. Designed to be clear, responsive, and cross-platform for Mac and Linux (amd64, arm64).

## Features
- **Offline Mode**: Displays current time and timezone for tracked/searched cities if internet is unavailable.
- **Online Mode**: Provides current date, time, and a 7-day weather forecast.
- **Hourly Outlook**: 12-hour trend panel with Unicode sparkline graphs for temperature, wind speed, and precipitation probability, shown next to the current weather.
- **Favorites**: Manage up to 5 favorite cities for quick reference.
- **Search capabilities**: Type `/` to search for cities worldwide. Features autocomplete suggestions after typing 3 characters.
- **Unit Configuration**: Press `u` to toggle temperature (Celsius/Fahrenheit) and wind speeds (km/h, m/s, mph).

## Installation

### Via Go
If you have Go 1.24+ installed on your system, you can easily install the TUI directly from this repository:
```bash
go install github.com/kubevoidcraft/weather-tui@latest
```
The application will be compiled into your `$GOPATH/bin/` directory and can be executed via `weather-tui`.

### Native Binaries
Pre-compiled executable binaries for `macOS` and `Linux` on both `amd64` and `arm64` architectures are automatically built and published under the **[Releases](https://github.com/kubevoidcraft/weather-tui/releases)** tab alongside every version tag!

## Technologies
- Written in **Go**.
- TUI powered by `tview` / `tcell`.
- Weather and Geocoding powered by [Open-Meteo](https://open-meteo.com/) API (no API key required).
- Checked by `golangci-lint`.

## Development

### Running tests

```bash
go test ./...
```

With coverage:

```bash
go test -cover ./...
# Or for a detailed report:
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

### CI

Every push and pull request runs the `CI` workflow (lint + `govulncheck` + tests with race detector + coverage report uploaded as an artifact). Releases are gated on a successful test run, so broken builds cannot reach the Releases page.

Coverage totals are printed in the CI log (`go tool cover -func`) and the full `coverage.out` / `coverage.txt` files are attached to each run as a downloadable artifact.

### Security posture

- **No API keys**: Open-Meteo is used unauthenticated, so there are no secrets to rotate or leak.
- **Vulnerability scanning**: `govulncheck` runs on every push / PR and fails the build when a reachable CVE is detected in the Go module graph.
- **Automated dependency updates**: Dependabot opens weekly grouped PRs for Go modules and GitHub Actions.
- **Strict HTTP client**: 10 s total timeout, `context.Context` cancellation, identifying `User-Agent`, and a 1 MiB response-body cap on every Open-Meteo call.
- **Rate-limit friendly**: the search autocomplete debounces keystrokes (250 ms) and cancels in-flight requests when the user keeps typing, so one word turns into one request — not one per character.
