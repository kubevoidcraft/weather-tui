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
	Header        *tview.TextView
	Sidebar       *tview.List
	MainView      *tview.TextView
	CmdInput      *tview.InputField
	cmdVisible    bool
	Config        *config.Config
	CurrentCity   *weather.City
	SearchList    *tview.List
	SearchResults []weather.City
}

// NewApp creates and initializes a new App instance with all UI components.
func NewApp() *App {
	cfg, err := config.Load()
	if err != nil {
		// Log or handle, for now we will just start with empty config if it totally fails
		cfg = &config.Config{}
	}

	// Material Design Theme Colors
	bgColor := tview.Styles.PrimitiveBackgroundColor
	primaryColor := tcell.ColorTeal

	header := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	header.SetBackgroundColor(primaryColor)
	header.SetTextColor(tcell.ColorWhite)

	sidebar := tview.NewList().ShowSecondaryText(true).SetMainTextColor(tcell.ColorWhite).SetSecondaryTextColor(tcell.ColorSilver).SetSelectedBackgroundColor(tcell.ColorDarkCyan)

	mainView := tview.NewTextView().SetDynamicColors(true).SetRegions(true).SetWordWrap(true)
	mainView.SetBackgroundColor(bgColor)

	cmdInput := tview.NewInputField().SetLabel(" 🔍 ").SetLabelColor(tcell.ColorYellow).SetFieldBackgroundColor(tcell.ColorDarkSlateGray)

	searchList := tview.NewList().ShowSecondaryText(true).SetMainTextColor(tcell.ColorWhite).SetSecondaryTextColor(tcell.ColorDarkGray).SetSelectedBackgroundColor(tcell.ColorTeal)

	a := &App{
		App:        tview.NewApplication(),
		Pages:      tview.NewPages(),
		MainFlex:   tview.NewFlex(), // No background color, inherits default
		Header:     header,
		Sidebar:    sidebar,
		MainView:   mainView,
		CmdInput:   cmdInput,
		Config:     cfg,
		SearchList: searchList,
	}
	a.setupUI()
	return a
}

func (a *App) setupUI() {
	// Sidebar setup
	a.Sidebar.SetTitle(" Favorites ").SetBorder(false)
	a.refreshFavorites()

	// Sidebar event handling
	a.Sidebar.SetSelectedFunc(func(index int, _, _ string, _ rune) {
		if index >= len(a.Config.Favorites) {
			return // The "Add favorite" hint or empty
		}
		// Set current city
		city := a.Config.Favorites[index]
		a.CurrentCity = &city

		a.MainView.SetText(fmt.Sprintf("\n[teal]  Loading forecast for %s...[-]", city.Label()))
		go a.fetchForecast(city)
	})

	// MainView setup
	a.MainView.SetTitle(" Forecast ").SetBorder(false)
	a.MainView.SetText("\n[teal]  Welcome to Weather TUI![-]\n\n  Press [yellow]'/'[-] to search for a city.\n  Press [yellow]'Tab'[-] to switch to Favorites.\n  Press [yellow]'q'[-] or Ctrl+C to quit.\n")

	// Header setup
	a.updateHeader()

	// Content layout: Sidebar (left) + MainView (right)
	contentFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(a.Sidebar, 20, 1, false).
		AddItem(a.MainView, 0, 3, true)

	// Root layout: Header + Content
	a.MainFlex.SetDirection(tview.FlexRow).
		AddItem(a.Header, 1, 1, false).
		AddItem(contentFlex, 0, 1, true)

	// Add to Pages
	a.Pages.AddPage("main", a.MainFlex, true, true)

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
		if event.Key() == tcell.KeyDown || event.Key() == tcell.KeyUp || event.Key() == tcell.KeyPgDn || event.Key() == tcell.KeyPgUp {
			if a.hasItem(a.MainFlex, a.SearchList) {
				a.App.SetFocus(a.SearchList)
				// Pass the event to the list so it immediately moves
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
			// We trigger forecast fetch here asynchronously
			a.MainView.SetText(fmt.Sprintf("\n[teal]  Loading forecast for %s...[-]", city.Label()))
			go a.fetchForecast(city)
		}
	})

	// Global Key Bindings
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

		// Navigate focus between Sidebar and MainView
		if event.Key() == tcell.KeyTab {
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

		return event
	})

	a.App.SetRoot(a.Pages, true).EnableMouse(true)

	// Start a goroutine to update the time in the header every second
	go func() {
		for {
			time.Sleep(1 * time.Second)
			a.App.QueueUpdateDraw(func() {
				a.updateHeader()
			})
		}
	}()
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
	forecast, err := weather.GetForecast(city.Lat, city.Lon, city.Timezone)
	a.App.QueueUpdateDraw(func() {
		if err != nil {
			a.MainView.SetText(fmt.Sprintf("Error fetching forecast: %v", err))
			return
		}

		// Update Header to show offline fallback vs Live Data?
		// Already handled by updateHeader since CurrentCity is set
		a.updateHeader()

		// Format output
		out := fmt.Sprintf("\n  [white::b]📍 %s[-]\n  [gray]Timezone: %s[-]\n\n", city.Label(), city.Timezone)
		out += "  [teal::b]Current Weather[-]\n"
		out += fmt.Sprintf("  🌡️  [white]%.1f°C[-]\n", forecast.Current.Temperature2m)
		out += fmt.Sprintf("  💨  [white]%.1f km/h[-]\n", forecast.Current.WindSpeed10m)
		out += fmt.Sprintf("  🌤️  [white]%s[-]\n\n", weather.WMOToText(forecast.Current.WeatherCode))

		out += "  [teal::b]7-Day Forecast[-]\n"
		for i, date := range forecast.Daily.Time {
			minT := forecast.Daily.Temperature2mMin[i]
			maxT := forecast.Daily.Temperature2mMax[i]
			cond := weather.WMOToText(forecast.Daily.WeatherCode[i])
			out += fmt.Sprintf("  [yellow]%s[-]  %-20s [blue]▼ %.1f°C[-]  [red]▲ %.1f°C[-]\n", date, cond, minT, maxT)
		}

		a.MainView.SetText(out)
	})
}

// refreshFavorites rebuilds the sidebar items from the Config.Favorites
func (a *App) refreshFavorites() {
	currentIndex := a.Sidebar.GetCurrentItem()
	a.Sidebar.Clear()
	for _, f := range a.Config.Favorites {
		// Show city label as main, local time as secondary
		timeStr := f.LocalTime().Format("15:04 (MST)")
		a.Sidebar.AddItem(" "+f.Label(), "   "+timeStr, 0, nil)
	}
	if len(a.Config.Favorites) == 0 {
		a.Sidebar.AddItem(" No Favorites", "   Press 'f' to add", 0, nil)
	}

	if currentIndex >= 0 && currentIndex < a.Sidebar.GetItemCount() {
		a.Sidebar.SetCurrentItem(currentIndex)
	}
}

func (a *App) updateHeader() {
	status := "Offline Mode / Default"
	if a.CurrentCity != nil {
		status = fmt.Sprintf("City: %s", a.CurrentCity.Label())
	}

	// App-wide sync time
	sysTime := time.Now().Format("2006-01-02 15:04:05 MST")
	a.Header.SetText(fmt.Sprintf(" %s | System Time: %s ", status, sysTime))

	// Also re-render the sidebar to keep favorite clocks ticking
	a.refreshFavorites()
}

// Run starts the application event loop.
func (a *App) Run() error {
	return a.App.Run()
}
