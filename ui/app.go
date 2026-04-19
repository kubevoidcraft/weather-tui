// Package ui contains the terminal user interface logic for weather-tui.
package ui

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kubevoidcraft/weather-tui/internal/config"
	"github.com/kubevoidcraft/weather-tui/internal/weather"
	"github.com/rivo/tview"
)

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
	if idx < 0 || idx >= len(a.Config.Favorites) {
		// Hide the selection bar by matching the default background color.
		a.Sidebar.SetSelectedBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
		a.Sidebar.SetSelectedTextColor(tcell.ColorWhite)
		return
	}
	a.Sidebar.SetSelectedBackgroundColor(tcell.ColorDarkCyan)
	a.Sidebar.SetSelectedTextColor(tcell.ColorWhite)
	a.Sidebar.SetCurrentItem(idx)
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

		// Determine unit labels based on config
		tempLabel := "°C"
		if a.Config.TemperatureUnit == "fahrenheit" {
			tempLabel = "°F"
		}

		windLabel := "km/h"
		switch a.Config.WindUnit {
		case "ms":
			windLabel = "m/s"
		case "mph":
			windLabel = "mph"
		}

		// Format output
		out := fmt.Sprintf("\n  [white::b]📍 %s[-]\n  [gray]Timezone: %s[-]\n\n", city.Label(), city.Timezone)
		out += "  [teal::b]Current Weather[-]\n"
		out += fmt.Sprintf("  🌡️  [white]%.1f%s[-]\n", forecast.Current.Temperature2m, tempLabel)
		out += fmt.Sprintf("  💨  [white]%.1f %s[-]\n", forecast.Current.WindSpeed10m, windLabel)
		out += fmt.Sprintf("  🌤️  [white]%s[-]\n\n", weather.WMOToText(forecast.Current.WeatherCode))

		out += "  [teal::b]7-Day Forecast[-]\n"
		for i, date := range forecast.Daily.Time {
			minT := forecast.Daily.Temperature2mMin[i]
			maxT := forecast.Daily.Temperature2mMax[i]
			cond := weather.WMOToText(forecast.Daily.WeatherCode[i])
			out += fmt.Sprintf("  [yellow]%s[-]  %-20s [blue]▼ %.1f%s[-]  [red]▲ %.1f%s[-]\n", date, cond, minT, tempLabel, maxT, tempLabel)
		}

		a.MainView.SetText(out)
	})
}

// refreshFavorites rebuilds the sidebar items from the Config.Favorites and
// re-applies the highlight for the currently active city (if it is a favorite).
func (a *App) refreshFavorites() {
	a.Sidebar.Clear()
	for i, f := range a.Config.Favorites {
		// Show city label as main, local time as secondary
		timeStr := f.LocalTime().Format("15:04 (MST)")
		a.Sidebar.AddItem(" "+f.Label(), "     "+timeStr, rune('1'+i), nil)
	}
	if len(a.Config.Favorites) == 0 {
		a.Sidebar.AddItem(" No Favorites", "   Press 'f' to add", 0, nil)
	}

	// Re-apply the highlight based on the currently active city. This ensures
	// that periodic redraws (for ticking clocks) or favorite list mutations
	// (add/remove) keep the sidebar visually consistent with app state.
	if a.CurrentCity != nil {
		a.highlightFavorite(a.favoriteIndex(*a.CurrentCity))
	} else {
		a.highlightFavorite(-1)
	}
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
