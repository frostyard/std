// Command deploy simulates a multi-step deployment pipeline to demonstrate
// the reporter package's Step, Message, Warning, and Complete methods.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/frostyard/std/reporter"
)

func main() {
	format := flag.String("format", "text", "output format: text, json, noop")
	flag.Parse()

	var r reporter.Reporter
	switch *format {
	case "text":
		r = reporter.NewTextReporter(os.Stdout)
	case "json":
		r = reporter.NewJSONReporter(os.Stdout)
	case "noop":
		r = reporter.NoopReporter{}
	default:
		fmt.Fprintf(os.Stderr, "unknown format: %s\n", *format)
		os.Exit(1)
	}

	// Step 1: Validate configuration
	r.Step(1, 4, "Validating configuration")
	time.Sleep(200 * time.Millisecond)
	r.Message("Config loaded from deploy.yaml")
	r.Warning("deprecated key 'legacy_mode' in config")

	// Step 2: Build
	r.Step(2, 4, "Building application")
	time.Sleep(300 * time.Millisecond)
	r.Message("Compiled 12 packages")

	// Step 3: Test
	r.Step(3, 4, "Running tests")
	time.Sleep(400 * time.Millisecond)
	r.Message("47 passed, 1 skipped")
	r.Warning("test coverage below 80%%")

	// Step 4: Deploy
	r.Step(4, 4, "Deploying to staging")
	time.Sleep(300 * time.Millisecond)
	r.Message("Deployed to staging-01.example.com")

	type deployDetails struct {
		Version     string `json:"version"`
		Environment string `json:"environment"`
		Host        string `json:"host"`
	}

	r.Complete("Deploy complete: v1.2.3 → staging", deployDetails{
		Version:     "v1.2.3",
		Environment: "staging",
		Host:        "staging-01.example.com",
	})
}
