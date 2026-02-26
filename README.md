# Weather TUI

A terminal-based interface (TUI) for weather forecasts. Designed to be clear, responsive, and cross-platform for Mac and Linux (amd64, arm64).

## Features
- **Offline Mode**: Displays current time and timezone for tracked/searched cities if internet is unavailable.
- **Online Mode**: Provides current date, time, and a 7-day weather forecast.
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
