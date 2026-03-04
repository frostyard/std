// Command healthcheck simulates checking multiple services to demonstrate
// the reporter package's Step, Message, MessagePlain, Warning, Error, and Complete methods.
package main

import (
	"errors"
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

	type check struct {
		name    string
		status  string // "healthy", "degraded", "down"
		detail  string
		latency string
	}

	checks := []check{
		{"database", "healthy", "PostgreSQL: connected", "12ms"},
		{"cache", "healthy", "Redis: connected", "2ms"},
		{"API gateway", "degraded", "response time degraded", "850ms"},
		{"message queue", "down", "unreachable", ""},
	}

	healthy, degraded, down := 0, 0, 0

	for i, c := range checks {
		r.Step(i+1, len(checks), fmt.Sprintf("Checking %s", c.name))
		time.Sleep(200 * time.Millisecond)

		switch c.status {
		case "healthy":
			r.Message("%s (latency %s)", c.detail, c.latency)
			healthy++
		case "degraded":
			r.Warning("%s %s (%s)", c.name, c.detail, c.latency)
			degraded++
		case "down":
			r.Error(errors.New("connection refused"), fmt.Sprintf("%s %s", c.name, c.detail))
			down++
		}
	}

	r.MessagePlain("") // blank line before summary in text mode
	r.Complete(
		fmt.Sprintf("Health check: %d healthy, %d degraded, %d down", healthy, degraded, down),
		nil,
	)
}
