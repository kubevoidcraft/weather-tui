// Package main is the entry point for the weather-tui application.
package main

import (
	"log"

	"github.com/kubevoidcraft/weather-tui/ui"
)

func main() {
	app := ui.NewApp()

	if err := app.Run(); err != nil {
		log.Fatalf("Error running application: %v\n", err)
	}
}
