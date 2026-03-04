# Reporter Example CLI Applications — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add four scenario-based example CLIs under `_examples/` that demonstrate idiomatic usage of the `reporter` package.

**Architecture:** Each example is a standalone `package main` with a `--format` flag to switch between text/json/noop reporters. The `_` prefix excludes them from `go build ./...`. All examples follow the same structure: parse flag, construct reporter, run scenario, exit.

**Tech Stack:** Go 1.26, stdlib only, `github.com/frostyard/std/reporter` package.

---

### Task 1: Create the fileprocess example

**Files:**
- Create: `_examples/fileprocess/main.go`

**Step 1: Create the directory**

Run: `mkdir -p _examples/fileprocess`

**Step 2: Write the example**

```go
// Command fileprocess simulates batch file processing to demonstrate
// the reporter package's Step, Progress, Message, and Complete methods.
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

	files := []struct {
		name  string
		pages int
		skip  bool
	}{
		{"document.pdf", 42, false},
		{"photo.jpg", 1, false},
		{"archive.tar.gz", 0, true},
		{"report.docx", 18, false},
		{"notes.txt", 3, true},
	}

	succeeded := 0
	skipped := 0

	for i, f := range files {
		r.Step(i+1, len(files), fmt.Sprintf("Scanning %s", f.name))
		time.Sleep(200 * time.Millisecond)

		if f.skip {
			r.Message("Skipped (unsupported format)")
			skipped++
			continue
		}

		r.Message("Found %d pages", f.pages)
		r.Progress((i+1)*100/len(files), fmt.Sprintf("%d%% complete", (i+1)*100/len(files)))
		succeeded++
	}

	r.Complete(
		fmt.Sprintf("Processed %d files (%d succeeded, %d skipped)", len(files), succeeded, skipped),
		nil,
	)

	if !r.IsJSON() {
		fmt.Println("\nTip: run with --format=json for machine-readable output")
	}
}
```

**Step 3: Verify it compiles and runs**

Run: `cd _examples/fileprocess && go run .`
Expected: text output showing 5 steps with file scanning messages and completion banner.

Run: `cd _examples/fileprocess && go run . --format=json`
Expected: JSON Lines output with step/message/progress/complete events.

Run: `cd _examples/fileprocess && go run . --format=noop`
Expected: no output (noop discards everything), plus no tip since `IsJSON()` is false.

Wait — noop returns false for `IsJSON()`, so the tip will still print. That's fine — it demonstrates the discriminator correctly (noop is not JSON, so the tip is relevant).

**Step 4: Commit**

```bash
git add _examples/fileprocess/main.go
git commit -m "feat: add fileprocess example for reporter package"
```

---

### Task 2: Create the deploy example

**Files:**
- Create: `_examples/deploy/main.go`

**Step 1: Create the directory**

Run: `mkdir -p _examples/deploy`

**Step 2: Write the example**

```go
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
```

**Step 3: Verify it compiles and runs**

Run: `cd _examples/deploy && go run .`
Expected: text output showing 4-step deploy pipeline with warnings and completion.

Run: `cd _examples/deploy && go run . --format=json`
Expected: JSON Lines with structured `details` field on the complete event.

**Step 4: Commit**

```bash
git add _examples/deploy/main.go
git commit -m "feat: add deploy example for reporter package"
```

---

### Task 3: Create the healthcheck example

**Files:**
- Create: `_examples/healthcheck/main.go`

**Step 1: Create the directory**

Run: `mkdir -p _examples/healthcheck`

**Step 2: Write the example**

```go
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
```

**Step 3: Verify it compiles and runs**

Run: `cd _examples/healthcheck && go run .`
Expected: text output with healthy/degraded/down statuses and completion summary.

Run: `cd _examples/healthcheck && go run . --format=json`
Expected: JSON Lines with error events containing `details.error` field.

**Step 4: Commit**

```bash
git add _examples/healthcheck/main.go
git commit -m "feat: add healthcheck example for reporter package"
```

---

### Task 4: Create the migration example

**Files:**
- Create: `_examples/migration/main.go`

**Step 1: Create the directory**

Run: `mkdir -p _examples/migration`

**Step 2: Write the example**

```go
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
```

**Step 3: Verify it compiles and runs**

Run: `cd _examples/migration && go run .`
Expected: text output with 3 phases, progress percentages in migrate phase, warning about skipped records.

Run: `cd _examples/migration && go run . --format=json`
Expected: JSON Lines with progress events showing percentage and structured completion details.

**Step 4: Commit**

```bash
git add _examples/migration/main.go
git commit -m "feat: add migration example for reporter package"
```

---

### Task 5: Final verification and cleanup

**Step 1: Run all examples in text mode**

```bash
for dir in _examples/*/; do
  echo "=== $(basename "$dir") ==="
  (cd "$dir" && go run .)
  echo
done
```

**Step 2: Run all examples in JSON mode**

```bash
for dir in _examples/*/; do
  echo "=== $(basename "$dir") ==="
  (cd "$dir" && go run . --format=json)
  echo
done
```

**Step 3: Run `make check` from the project root**

Run: `make check`
Expected: all existing tests pass, lint clean. The `_examples/` directory should be excluded from `go build ./...` by the `_` prefix.

**Step 4: Verify `go vet` on examples**

```bash
for dir in _examples/*/; do
  echo "=== $(basename "$dir") ==="
  (cd "$dir" && go vet .)
done
```

Expected: no issues.
