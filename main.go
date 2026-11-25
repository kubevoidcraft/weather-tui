// Package main is the entry point for the weather-tui application.
package main

import (
	"flag"
	"fmt"
	"log"
	"runtime/debug"

	"github.com/kubevoidcraft/weather-tui/ui"
)

// Build metadata. Release binaries get these via GoReleaser ldflags
// (-X main.version=...); see .goreleaser.yaml. Package-level vars are
// required for -X injection and are never mutated at runtime.
var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(versionString(version, commit, date, readBuildVersion()))
		return
	}

	app := ui.NewApp()

	if err := app.Run(); err != nil {
		log.Fatalf("Error running application: %v\n", err)
	}
}

// readBuildVersion returns the module version recorded by the Go toolchain,
// which is set for `go install ...@vX.Y.Z` builds that bypass GoReleaser.
func readBuildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return info.Main.Version
}

// versionString formats the -version output. ldflags-injected values win;
// otherwise it falls back to the module version from the build info.
func versionString(ver, commit, date, buildVersion string) string {
	if ver == "dev" && buildVersion != "" && buildVersion != "(devel)" {
		ver = buildVersion
	}
	s := "weather-tui " + ver
	if commit != "" {
		s += fmt.Sprintf(" (commit %s, built %s)", commit, date)
	}
	return s
}
