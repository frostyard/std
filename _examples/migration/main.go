// Command migration simulates a data migration to demonstrate
// the reporter package's Step, Progress, Message, Warning, and Complete methods.
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

	const (
		totalRecords = 100
		batchSize    = 20
		batches      = totalRecords / batchSize
		skipped      = 3
	)

	// Phase 1: Validate
	r.Step(1, 3, "Validating source data")
	time.Sleep(200 * time.Millisecond)
	r.Message("Found %d records to migrate", totalRecords)

	// Phase 2: Migrate in batches
	r.Step(2, 3, "Migrating records")
	for batch := range batches {
		time.Sleep(200 * time.Millisecond)
		pct := (batch + 1) * 100 / batches
		r.Progress(pct, fmt.Sprintf("batch %d of %d complete", batch+1, batches))

		// Simulate skipped records in batch 3
		if batch == 2 {
			r.Warning("%d records skipped (missing required field)", skipped)
		}
	}

	// Phase 3: Verify
	r.Step(3, 3, "Verifying migration")
	time.Sleep(200 * time.Millisecond)
	migrated := totalRecords - skipped
	r.Message("%d records verified", migrated)

	type migrationStats struct {
		Migrated int `json:"migrated"`
		Skipped  int `json:"skipped"`
		Failed   int `json:"failed"`
	}

	r.Complete(
		fmt.Sprintf("Migration complete: %d migrated, %d skipped, 0 failed", migrated, skipped),
		migrationStats{Migrated: migrated, Skipped: skipped, Failed: 0},
	)
}
