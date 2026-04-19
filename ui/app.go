// Package ui contains the terminal user interface logic for weather-tui.
package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kubevoidcraft/weather-tui/internal/config"
	"github.com/kubevoidcraft/weather-tui/internal/weather"
	"github.com/rivo/tview"
)

// hourlyWindowHours is how many hours of hourly forecast we render in the
// side panel. Twelve fits comfortably next to the current-weather block on a
// standard terminal and still shows a meaningful trend.
const hourlyWindowHours = 12

// sideBySideMinWidth is the minimum MainView inner width at which the hourly
// panel is rendered next to the current weather block. Below this threshold,
// the hourly panel is stacked underneath instead so nothing gets clipped.
const sideBySideMinWidth = 80

// currentWeatherColumnWidth is the fixed render width of the left (current
// weather) column when the layout is side-by-side. It is wide enough for a
// typical city label while leaving room for the hourly panel.
const currentWeatherColumnWidth = 32

// App represents the main application state and UI components.
type App struct {
	App           *tview.Application
	Pages         *tview.Pages
	MainFlex      *tview.Flex
	HeaderFlex    *tview.Flex
	HeaderInfo    *tview.TextView
	Sidebar       *tview.List
	MainView      *tview.TextView
	CmdInput      *tview.InputField
	cmdVisible    bool
	Config        *config.Config
	CurrentCity   *weather.City
	SearchList    *tview.List
	SearchResults []weather.City
	SettingsForm  *tview.Form
}

const asciiArt = `
  [teal] _ _ _         _   _              [yellow]_____ _   _ _____ 
  [teal]| | | |___ ___| |_| |_ ___ ___    [yellow]|_   _| | | |     |
  [teal]| | | | -_| .'|  _|   | -_|  _|     [yellow]| | | |_| |-   -|
  [teal]|_____|___|__,|_| |_|_|___|_|       [yellow]|_| |___|_____|[-]

  Press [yellow]'/'[-] to search        Press [yellow]'f'[-] to toggle favorite
  Press [yellow]'Tab'[-] to switch        Press [yellow]'1'-'5'[-] to load favorite
  Press [yellow]'u'[-] to config units      Press [yellow]'q'[-] or Ctrl+C to quit
`

// NewApp creates and initializes a new App instance with all UI components.
func NewApp() *App {
	cfg, err := config.Load()
	if err != nil {
		// Log or handle, for now we will just start with empty config if it totally fails
		cfg = &config.Config{}
	}

	// Material Design Theme Colors
	bgColor := tview.Styles.PrimitiveBackgroundColor

	headerInfo := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	headerInfo.SetBackgroundColor(bgColor)

	headerFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(headerInfo, 0, 1, false)

	sidebar := tview.NewList().ShowSecondaryText(true).SetMainTextColor(tcell.ColorWhite).SetSecondaryTextColor(tcell.ColorSilver)
	// Keep the selection highlight visible even when the sidebar is not focused,
	// so users can always see which favorite is currently loaded.
	sidebar.SetSelectedFocusOnly(false)

	mainView := tview.NewTextView().SetDynamicColors(true).SetRegions(true).SetWordWrap(true)
	mainView.SetBackgroundColor(bgColor)

	cmdInput := tview.NewInputField().SetLabel(" 🔍 ").SetLabelColor(tcell.ColorYellow).SetFieldBackgroundColor(tcell.ColorDarkSlateGray)

	searchList := tview.NewList().ShowSecondaryText(true).SetMainTextColor(tcell.ColorWhite).SetSecondaryTextColor(tcell.ColorDarkGray).SetSelectedBackgroundColor(tcell.ColorTeal)

	settingsForm := tview.NewForm()
	settingsForm.SetBorder(true).SetTitle(" Preferences ").SetTitleColor(tcell.ColorTeal).SetBorderColor(tcell.ColorTeal)

	a := &App{
		App:          tview.NewApplication(),
		Pages:        tview.NewPages(),
		MainFlex:     tview.NewFlex(), // No background color, inherits default
		HeaderFlex:   headerFlex,
		HeaderInfo:   headerInfo,
		Sidebar:      sidebar,
		MainView:     mainView,
		CmdInput:     cmdInput,
		Config:       cfg,
		SearchList:   searchList,
		SettingsForm: settingsForm,
	}
	a.setupUI()
	return a
}

func (a *App) setupUI() {
	// Sidebar setup
	a.Sidebar.SetTitle(" Favorites ").SetBorder(false)
	a.refreshFavorites()
	// Nothing is selected at startup - hide the selection highlight until the
	// user activates a favorite (via hotkey, sidebar Enter or search match).
	a.highlightFavorite(-1)

	// Sidebar event handling
	a.Sidebar.SetSelectedFunc(func(index int, _, _ string, _ rune) {
		if index >= len(a.Config.Favorites) {
			return // The "Add favorite" hint or empty
		}
		// Set current city
		city := a.Config.Favorites[index]
		a.CurrentCity = &city
		a.highlightFavorite(index)

		a.MainView.SetText(fmt.Sprintf("\n[teal]  Loading forecast for %s...[-]", city.Label()))
		go a.fetchForecast(city)
	})

	// MainView setup
	a.MainView.SetTitle(" Forecast ").SetBorder(false)
	a.MainView.SetText("")

	// Header setup
	a.updateHeader()

	// Content layout: Sidebar (left) + MainView (right) (25% / 75%)
	contentFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(a.Sidebar, 0, 1, false).
		AddItem(a.MainView, 0, 3, true)

	// Root layout: Header + Content (40% / 60%)
	a.MainFlex.SetDirection(tview.FlexRow).
		AddItem(a.HeaderFlex, 0, 4, false).
		AddItem(contentFlex, 0, 6, true)

	// Add to Pages
	a.Pages.AddPage("main", a.MainFlex, true, true)

	// Settings Form Setup
	tempOptions := []string{"celsius", "fahrenheit"}
	windOptions := []string{"kmh", "ms", "mph"}

	tempIdx := 0
	if a.Config.TemperatureUnit == "fahrenheit" {
		tempIdx = 1
	}

	windIdx := 0
	switch a.Config.WindUnit {
	case "ms":
		windIdx = 1
	case "mph":
		windIdx = 2
	}

	// Temporary variables for holding selection until "Save" is clicked
	selectedTemp := a.Config.TemperatureUnit
	selectedWind := a.Config.WindUnit

	a.SettingsForm.SetButtonBackgroundColor(tcell.ColorDarkCyan)
	a.SettingsForm.SetButtonTextColor(tcell.ColorWhite)
	a.SettingsForm.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	a.SettingsForm.SetFieldTextColor(tcell.ColorWhite)
	a.SettingsForm.SetLabelColor(tcell.ColorTeal)

	a.SettingsForm.AddDropDown("Temperature", tempOptions, tempIdx, func(option string, _ int) {
		selectedTemp = option
	})

	a.SettingsForm.AddDropDown("Wind Speed", windOptions, windIdx, func(option string, _ int) {
		selectedWind = option
	})

	a.SettingsForm.AddButton("Save", func() {
		a.Config.TemperatureUnit = selectedTemp
		a.Config.WindUnit = selectedWind
		_ = a.Config.Save()
		if a.CurrentCity != nil {
			go a.fetchForecast(*a.CurrentCity)
		}
		a.Pages.HidePage("settings")
		a.App.SetFocus(a.MainView)
	})

	a.SettingsForm.AddButton("Close", func() {
		// Just hide, don't save
		a.Pages.HidePage("settings")
		a.App.SetFocus(a.MainView)
	})

	a.SettingsForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			a.Pages.HidePage("settings")
			a.App.SetFocus(a.MainView)
			return nil
		}

		// Check if a dropdown is currently focused and open
		itemIndex, _ := a.SettingsForm.GetFocusedItemIndex()
		if itemIndex >= 0 && itemIndex < a.SettingsForm.GetFormItemCount() {
			item := a.SettingsForm.GetFormItem(itemIndex)
			if dd, ok := item.(*tview.DropDown); ok && dd.IsOpen() {
				// Let the dropdown handle up/down natively when open
				return event
			}
		}

		// Map up/down arrows to tab/backtab to navigate between form fields
		// rather than manipulating the dropdown list values immediately
		if event.Key() == tcell.KeyDown {
			return tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
		}
		if event.Key() == tcell.KeyUp {
			return tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModNone)
		}

		return event
	})

	// Center the settings form
	settingsFlex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(a.SettingsForm, 11, 1, true).
			AddItem(nil, 0, 1, false),
			40, 1, true).
		AddItem(nil, 0, 1, false)

	a.Pages.AddPage("settings", settingsFlex, true, false)

	// Command Input handling
	a.SearchList.SetTitle(" Suggestions ").SetBorder(true).SetBorderColor(tcell.ColorTeal)

	a.CmdInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			a.hideCommandInput()
		case tcell.KeyEnter:
			if a.hasItem(a.MainFlex, a.SearchList) {
				// Focus the search list instead
				a.App.SetFocus(a.SearchList)
			} else {
				a.hideCommandInput()
			}
		}
	})

	a.CmdInput.SetChangedFunc(func(text string) {
		// Just expecting the query text directly now, no "s " prefix
		if len(text) >= 3 {
			go a.executeSearch(text)
		} else {
			a.hideSearchList()
		}
	})

	a.CmdInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// When the user presses arrow/page keys from inside the search box and
		// suggestions are visible, hand the key off to the SearchList instead
		// of letting the InputField consume it. We first move focus, then
		// manually invoke the list's input handler with the same event so the
		// selection updates immediately (without waiting for the next keypress).
		// The nil setFocus callback is safe because tview only uses it to
		// request focus changes, which we have already performed above.
		if event.Key() == tcell.KeyDown || event.Key() == tcell.KeyUp || event.Key() == tcell.KeyPgDn || event.Key() == tcell.KeyPgUp {
			if a.hasItem(a.MainFlex, a.SearchList) {
				a.App.SetFocus(a.SearchList)
				a.SearchList.InputHandler()(event, nil)
				return nil
			}
		}
		return event
	})

	a.SearchList.SetSelectedFunc(func(index int, _, _ string, _ rune) {
		if index >= 0 && index < len(a.SearchResults) {
			city := a.SearchResults[index]
			a.CurrentCity = &city
			a.hideCommandInput()
			// If the chosen city is already a favorite, highlight it in the
			// sidebar. Otherwise clear any stale highlight from a previous
			// selection so the sidebar does not falsely imply one is active.
			a.highlightFavorite(a.favoriteIndex(city))
			// We trigger forecast fetch here asynchronously
			a.MainView.SetText(fmt.Sprintf("\n[teal]  Loading forecast for %s...[-]", city.Label()))
			go a.fetchForecast(city)
		}
	})

	// Global Key Bindings
	a.setupGlobalKeybindings()

	a.App.SetRoot(a.Pages, true).EnableMouse(true)

	// Start a goroutine to update the time in the header every second.
	// updateHeader also re-renders the sidebar, which in turn re-applies the
	// current favorite highlight, so the UI remains consistent over time.
	go func() {
		for {
			time.Sleep(1 * time.Second)
			a.App.QueueUpdateDraw(func() {
				a.updateHeader()
			})
		}
	}()
}

func (a *App) setupGlobalKeybindings() {
	a.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// If command input is already focused, let it handle the event
		if a.App.GetFocus() == a.CmdInput {
			return event
		}

		// Toggle Command Bar on '/'
		if event.Rune() == '/' {
			if !a.cmdVisible {
				// We don't want it expanding, we want to fix it to exactly 1 line height
				a.MainFlex.AddItem(a.CmdInput, 1, 0, false)
				a.cmdVisible = true
				a.CmdInput.SetText("")
				a.App.SetFocus(a.CmdInput)
			}
			return nil
		}

		// Global quit
		if event.Rune() == 'q' {
			a.App.Stop()
			return nil
		}

		// Open settings modal
		if event.Rune() == 'u' {
			a.Pages.ShowPage("settings")
			a.SettingsForm.SetFocus(0)
			a.App.SetFocus(a.SettingsForm)
			return nil
		}

		// Navigate focus between Sidebar and MainView
		if event.Key() == tcell.KeyTab || event.Key() == tcell.KeyBacktab {
			// Do not steal tab if settings page is frontmost
			if frontPage, _ := a.Pages.GetFrontPage(); frontPage == "settings" {
				return event
			}

			if a.App.GetFocus() == a.Sidebar {
				a.App.SetFocus(a.MainView)
			} else {
				a.App.SetFocus(a.Sidebar)
			}
			return nil
		}

		// Add/remove favorite
		if event.Rune() == 'f' && a.CurrentCity != nil {
			isFav := false
			for _, fav := range a.Config.Favorites {
				if fav.Name == a.CurrentCity.Name {
					isFav = true
					break
				}
			}
			if isFav {
				a.Config.RemoveFavorite(a.CurrentCity.Name)
			} else {
				a.Config.AddFavorite(*a.CurrentCity)
			}
			_ = a.Config.Save()
			a.refreshFavorites()
			return nil
		}

		// Load favorite 1-5
		if event.Rune() >= '1' && event.Rune() <= '5' {
			idx := int(event.Rune() - '1')
			if idx < len(a.Config.Favorites) {
				city := a.Config.Favorites[idx]
				a.CurrentCity = &city
				// Highlight the selected favorite in the sidebar and move focus
				// there so the selection is visibly indicated to the user.
				a.highlightFavorite(idx)
				a.App.SetFocus(a.Sidebar)
				a.MainView.SetText(fmt.Sprintf("\n[teal]  Loading forecast for %s...[-]", city.Label()))
				go a.fetchForecast(city)
			}
			return nil
		}

		return event
	})
}

// favoriteIndex returns the index of the given city in Config.Favorites, or -1
// if the city is not currently saved as a favorite. Cities are matched by
// coordinates so duplicate names in different locations are disambiguated.
//
// Exact float equality is intentional: favorites are persisted with the same
// Lat/Lon values returned by Open-Meteo's geocoder, so as long as the geocoder
// is deterministic for a given city the comparison holds. If we ever observe
// drift, switch to a small epsilon tolerance here.
func (a *App) favoriteIndex(city weather.City) int {
	for i, fav := range a.Config.Favorites {
		if fav.Lat == city.Lat && fav.Lon == city.Lon {
			return i
		}
	}
	return -1
}

// highlightFavorite visually marks the favorite at idx as the currently active
// one. If idx is negative, any existing highlight is removed. The sidebar's
// current item still gets moved (tview always has one) but the selected-row
// background is only made visible when a real favorite is active.
func (a *App) highlightFavorite(idx int) {
	active := idx >= 0 && idx < len(a.Config.Favorites)
	a.applyHighlightColor(active)
	if active {
		a.Sidebar.SetCurrentItem(idx)
	}
}

func (a *App) hasItem(flex *tview.Flex, item tview.Primitive) bool {
	for i := 0; i < flex.GetItemCount(); i++ {
		if flex.GetItem(i) == item {
			return true
		}
	}
	return false
}

func (a *App) hideCommandInput() {
	a.hideSearchList()
	a.MainFlex.RemoveItem(a.CmdInput)
	a.cmdVisible = false
	a.CmdInput.SetText("")
	a.App.SetFocus(a.MainView)
}

func (a *App) hideSearchList() {
	if a.hasItem(a.MainFlex, a.SearchList) {
		// Assume CmdInput is the last item, we remove SearchList from above it
		a.MainFlex.RemoveItem(a.SearchList)
	}
}

func (a *App) executeSearch(query string) {
	results, err := weather.SearchCities(query)
	a.App.QueueUpdateDraw(func() {
		if err != nil {
			a.SearchList.Clear()
			a.SearchList.AddItem("Error finding city", err.Error(), 0, nil)
			if !a.hasItem(a.MainFlex, a.SearchList) {
				// Insert before command link
				a.MainFlex.AddItem(a.SearchList, 3, 0, false)
			}
			return
		}

		a.SearchResults = results
		a.SearchList.Clear()
		for i, r := range results {
			if i >= 10 { // Limit suggestions
				break
			}
			a.SearchList.AddItem(r.Label(), fmt.Sprintf("Lat: %.2f, Lon: %.2f", r.Lat, r.Lon), 0, nil)
		}

		if len(results) > 0 {
			if !a.hasItem(a.MainFlex, a.SearchList) {
				// Display enough rows strictly for the results + border padding
				a.MainFlex.AddItem(a.SearchList, len(results)+2, 0, false)
			} else {
				a.MainFlex.ResizeItem(a.SearchList, len(results)+2, 0)
			}
		} else {
			a.hideSearchList()
		}
	})
}

func (a *App) fetchForecast(city weather.City) {
	forecast, err := weather.GetForecast(city.Lat, city.Lon, city.Timezone, a.Config.TemperatureUnit, a.Config.WindUnit)
	a.App.QueueUpdateDraw(func() {
		if err != nil {
			a.MainView.SetText(fmt.Sprintf("Error fetching forecast: %v", err))
			return
		}

		a.updateHeader()

		tempLabel, windLabel := a.unitLabels()

		// Header line identifying the city being shown.
		out := fmt.Sprintf("\n  [white::b]📍 %s[-]\n  [gray]Timezone: %s[-]\n\n", city.Label(), city.Timezone)

		// The current weather and hourly forecast are rendered as two columns
		// when there is enough horizontal space; otherwise they stack.
		currentLines := renderCurrentWeather(forecast, tempLabel, windLabel)
		hourlyLines := renderHourlyPanel(forecast, city, tempLabel, windLabel)

		out += a.joinPanels(currentLines, hourlyLines)
		out += "\n"

		out += renderDailyForecast(forecast, tempLabel, windLabel)

		a.MainView.SetText(out)
	})
}

// renderDailyForecast formats the 7-day outlook as a titled table. A header
// row with column names sits above the data rows; both use the same fixed
// column widths so values line up cleanly under their labels. Widths are
// chosen to fit typical values plus a small margin.
func renderDailyForecast(forecast *weather.Forecast, tempLabel, windLabel string) string {
	const (
		dateWidth = 10 // "2006-01-02"
		condWidth = 20 // Weather description
		tempWidth = 10 // "▼ 12.3°C " with room for the arrow + decimals + unit
		windWidth = 14 // "💨 12.3 km/h" - emoji counts as 2 cols
	)

	var b strings.Builder
	b.WriteString("  [teal::b]7-Day Forecast[-]\n")
	// Header row: column titles in a muted color so they are clearly distinct
	// from data rows. padVisible handles wide characters in titles (none here,
	// but using it keeps the approach uniform with the rest of the renderer).
	b.WriteString("  ")
	b.WriteString(padVisible("[gray::b]Date[-]", dateWidth))
	b.WriteString("  ")
	b.WriteString(padVisible("[gray::b]Condition[-]", condWidth))
	b.WriteString(padVisible("[gray::b]Min[-]", tempWidth))
	b.WriteString(padVisible("[gray::b]Max[-]", tempWidth))
	b.WriteString(padVisible("[gray::b]Wind[-]", windWidth))
	b.WriteByte('\n')

	for i, date := range forecast.Daily.Time {
		minT := forecast.Daily.Temperature2mMin[i]
		maxT := forecast.Daily.Temperature2mMax[i]
		cond := weather.WMOToText(forecast.Daily.WeatherCode[i])
		wind := dailyWind(forecast.Daily.WindSpeed10mMax, i)

		b.WriteString("  ")
		b.WriteString(padVisible("[yellow]"+date+"[-]", dateWidth))
		b.WriteString("  ")
		b.WriteString(padVisible(cond, condWidth))
		b.WriteString(padVisible(fmt.Sprintf("[blue]▼ %.1f%s[-]", minT, tempLabel), tempWidth))
		b.WriteString(padVisible(fmt.Sprintf("[red]▲ %.1f%s[-]", maxT, tempLabel), tempWidth))
		if wind >= 0 {
			fmt.Fprintf(&b, "[white]💨 %.1f %s[-]", wind, windLabel)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// dailyWind safely returns the wind speed for day i, or -1 if the API did not
// include wind data (older response shapes or schema changes). Callers skip
// the wind column entirely on a negative return.
func dailyWind(winds []float64, i int) float64 {
	if i < 0 || i >= len(winds) {
		return -1
	}
	return winds[i]
}

// unitLabels returns display labels for temperature and wind speed units based
// on the currently configured preferences.
func (a *App) unitLabels() (tempLabel, windLabel string) {
	tempLabel = "°C"
	if a.Config.TemperatureUnit == "fahrenheit" {
		tempLabel = "°F"
	}

	windLabel = "km/h"
	switch a.Config.WindUnit {
	case "ms":
		windLabel = "m/s"
	case "mph":
		windLabel = "mph"
	}
	return tempLabel, windLabel
}

// renderCurrentWeather builds the "Current Weather" block as a slice of lines
// (one string per row). Returning lines rather than a pre-joined string keeps
// zipping with the hourly panel straightforward.
func renderCurrentWeather(forecast *weather.Forecast, tempLabel, windLabel string) []string {
	return []string{
		"  [teal::b]Current Weather[-]",
		fmt.Sprintf("  🌡️  [white]%.1f%s[-]", forecast.Current.Temperature2m, tempLabel),
		fmt.Sprintf("  💨  [white]%.1f %s[-]", forecast.Current.WindSpeed10m, windLabel),
		fmt.Sprintf("  🌤️  [white]%s[-]", weather.WMOToText(forecast.Current.WeatherCode)),
	}
}

// renderHourlyPanel produces the "Next N hours" block: a header row with hour
// labels followed by three sparkline rows (temperature, wind, precipitation
// probability). Each sparkline is annotated with a "first -> max -> last"
// summary so absolute values remain visible even though the block characters
// only show relative shape.
func renderHourlyPanel(forecast *weather.Forecast, city weather.City, tempLabel, windLabel string) []string {
	start := hourlyStartIndex(forecast.Hourly.Time, city)
	if start < 0 || start >= len(forecast.Hourly.Time) {
		return nil
	}

	end := start + hourlyWindowHours
	if end > len(forecast.Hourly.Time) {
		end = len(forecast.Hourly.Time)
	}

	times := forecast.Hourly.Time[start:end]
	temps := forecast.Hourly.Temperature2m[start:end]
	winds := forecast.Hourly.WindSpeed10m[start:end]
	rains := forecast.Hourly.PrecipitationProbability[start:end]

	// Header row: hour-of-day (HH) per column, separated by a single space so
	// it visually aligns one block character under each pair of digits.
	hourLabels := make([]string, 0, len(times))
	for _, t := range times {
		hourLabels = append(hourLabels, hourOfDay(t))
	}

	// Each hour occupies a fixed 3-column slot so the header labels and the
	// block characters below share the same column grid:
	//   header : "HH HH HH ..." (2 digits + 1 space per hour)
	//   data   : "B  B  B  ..." (1 block + 2 spaces per hour)
	// This way block i always sits directly under the first digit of hour
	// label i. labelPrefix and dataPrefix are sized so the two grids start
	// at the same terminal column regardless of emoji width differences.
	const labelPrefix = "  [gray]Time[-]  " // "  Time  " = 8 visible cols
	tempPrefix := padVisible("  🌡️  ", 8)
	windPrefix := padVisible("  💨  ", 8)
	rainPrefix := padVisible("  🌧️  ", 8)

	return []string{
		fmt.Sprintf("  [teal::b]Next %d hours[-]", len(times)),
		labelPrefix + strings.Join(hourLabels, " "),
		tempPrefix + spaceOut3(sparkline(temps)) + "  " + summarize(temps, tempLabel, "%.0f"),
		windPrefix + spaceOut3(sparkline(winds)) + "  " + summarize(winds, windLabel, "%.0f"),
		rainPrefix + spaceOut3(shaded(rains)) + "  " + summarize(rains, "%%", "%.0f"),
	}
}

// spaceOut3 renders a data row so each block sits in a 3-column slot,
// matching the "HH " grid of the header row. Blocks are separated by two
// spaces and the row has no trailing padding.
func spaceOut3(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return ""
	}
	parts := make([]string, len(runes))
	for i, r := range runes {
		parts[i] = string(r)
	}
	return strings.Join(parts, "  ")
}



// hourlyStartIndex returns the index of the first entry in the API's hourly
// Time slice that is at or after "now" in the city's local timezone. If the
// API didn't return enough data, it returns -1.
//
// Open-Meteo returns hourly data starting at 00:00 local time of the current
// day (when timezone is set), so we just need to find the first entry that is
// not in the past.
func hourlyStartIndex(times []string, city weather.City) int {
	if len(times) == 0 {
		return -1
	}
	now := city.LocalTime()
	// Trim to the beginning of the current hour to avoid skipping "this hour".
	now = now.Truncate(time.Hour)

	loc := now.Location()
	for i, raw := range times {
		// Open-Meteo hourly timestamps are of the form "2006-01-02T15:04" in
		// local time when the timezone query parameter was supplied.
		t, err := time.ParseInLocation("2006-01-02T15:04", raw, loc)
		if err != nil {
			continue
		}
		if !t.Before(now) {
			return i
		}
	}
	return -1
}

// hourOfDay extracts the two-digit hour component from an Open-Meteo hourly
// timestamp. If the string is malformed we fall back to "??" so the row still
// lines up with the sparkline cells.
func hourOfDay(raw string) string {
	if len(raw) < 13 {
		return "??"
	}
	return raw[11:13]
}

// summarize returns a "first -> max -> last unit" string for a numeric series.
// The %% case is handled by passing unit="%%" which fmt will render as "%".
func summarize(values []float64, unit, numFmt string) string {
	if len(values) == 0 {
		return ""
	}
	first, last := values[0], values[len(values)-1]
	maxV := values[0]
	for _, v := range values[1:] {
		if v > maxV {
			maxV = v
		}
	}
	format := fmt.Sprintf("[gray]%s → %s → %s %s[-]", numFmt, numFmt, numFmt, unit)
	return fmt.Sprintf(format, first, maxV, last)
}

// joinPanels combines the current-weather and hourly panels into a single
// multi-line string. It uses a side-by-side layout when MainView is wide
// enough, and falls back to stacking (current weather, blank line, hourly)
// on narrow terminals so nothing gets visually clipped.
func (a *App) joinPanels(left, right []string) string {
	if len(right) == 0 {
		return strings.Join(left, "\n") + "\n"
	}

	_, _, width, _ := a.MainView.GetInnerRect()
	if width > 0 && width < sideBySideMinWidth {
		var b strings.Builder
		for _, line := range left {
			b.WriteString(line)
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
		for _, line := range right {
			b.WriteString(line)
			b.WriteByte('\n')
		}
		return b.String()
	}

	rows := len(left)
	if len(right) > rows {
		rows = len(right)
	}
	var b strings.Builder
	for i := 0; i < rows; i++ {
		var l string
		if i < len(left) {
			l = left[i]
		}
		// Pad the left column to a fixed visible width so the right column
		// starts at a predictable position regardless of row content.
		b.WriteString(padVisible(l, currentWeatherColumnWidth))
		if i < len(right) {
			b.WriteString(right[i])
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// padVisible right-pads s with spaces until its visible width (ignoring tview
// style tags) reaches width. If s is already wider, it is returned unchanged.
// This keeps column alignment correct even when lines contain color markup.
func padVisible(s string, width int) string {
	visible := tview.TaggedStringWidth(s)
	if visible >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visible)
}

// refreshFavorites rebuilds the sidebar items from the Config.Favorites and
// re-applies the highlight for the currently active city (if it is a favorite).
//
// The cursor position is preserved so periodic redraws (driven by the clock
// ticker) do not snap the user back to the active favorite while they are
// navigating with arrow keys. The "active favorite" is conveyed purely via
// the selected-background color, which is bound to the list's current item;
// to keep that marker tied to the city being viewed rather than where the
// user's cursor happens to sit, we re-point the cursor at the active favorite
// only when the user is not currently focused on the sidebar.
func (a *App) refreshFavorites() {
	prevIndex := a.Sidebar.GetCurrentItem()

	a.Sidebar.Clear()
	for i, f := range a.Config.Favorites {
		// Show city label as main, local time as secondary
		timeStr := f.LocalTime().Format("15:04 (MST)")
		a.Sidebar.AddItem(" "+f.Label(), "     "+timeStr, rune('1'+i), nil)
	}
	if len(a.Config.Favorites) == 0 {
		a.Sidebar.AddItem(" No Favorites", "   Press 'f' to add", 0, nil)
	}

	activeIdx := -1
	if a.CurrentCity != nil {
		activeIdx = a.favoriteIndex(*a.CurrentCity)
	}

	// Keep the selection-bar color in sync with whether any favorite is active,
	// but choose the cursor position based on focus: if the user is navigating
	// the sidebar, preserve their cursor; otherwise snap the cursor (and thus
	// the visible highlight) to the active favorite.
	if a.App.GetFocus() == a.Sidebar {
		a.applyHighlightColor(activeIdx >= 0)
		if prevIndex >= 0 && prevIndex < a.Sidebar.GetItemCount() {
			a.Sidebar.SetCurrentItem(prevIndex)
		}
	} else {
		a.highlightFavorite(activeIdx)
	}
}

// applyHighlightColor toggles the sidebar's selected-row colors so the
// selection bar is only visible when a favorite is actively loaded. This is
// the color half of highlightFavorite, without moving the cursor.
func (a *App) applyHighlightColor(active bool) {
	if !active {
		a.Sidebar.SetSelectedBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
		a.Sidebar.SetSelectedTextColor(tcell.ColorWhite)
		return
	}
	a.Sidebar.SetSelectedBackgroundColor(tcell.ColorDarkCyan)
	a.Sidebar.SetSelectedTextColor(tcell.ColorWhite)
}

func (a *App) updateHeader() {
	sysTime := time.Now().Format("2006-01-02 15:04:05 MST")
	a.HeaderInfo.SetText(fmt.Sprintf("\n  [white::b]Weather TUI Dashboard[-]\n  [gray]System Time: %s[-]\n%s", sysTime, asciiArt))

	// Also re-render the sidebar to keep favorite clocks ticking
	a.refreshFavorites()
}

// Run starts the application event loop.
func (a *App) Run() error {
	return a.App.Run()
}
