package main

import (
	"fmt"
	"os"

	"github.com/itda-skills/cron-jobs/internal/app"
)

func main() {
	settings := app.LoadSettingsFromEnv()

	if err := settings.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "invalid settings: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("cron-jobs starting on %s with config %s\n", settings.Addr, settings.ConfigPath)
}
