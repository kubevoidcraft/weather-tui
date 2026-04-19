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

Every push and pull request runs the `CI` workflow (lint + tests with race detector + coverage report uploaded as an artifact). Releases are gated on a successful test run, so broken builds cannot reach the Releases page.

Coverage totals are printed in the CI log (`go tool cover -func`) and the full `coverage.out` / `coverage.txt` files are attached to each run as a downloadable artifact.
