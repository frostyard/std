# clix Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build `github.com/frostyard/clix`, a shared CLI convenience module that eliminates duplicated fang/cobra boilerplate across frostyard tools.

**Architecture:** Single flat Go package (`clix`) with four source files: app bootstrap (`clix.go`), common flags (`flags.go`), JSON output helpers (`output.go`), and reporter factory (`reporter.go`). Config struct pattern — callers create a `clix.App{}` and call `Run()`.

**Tech Stack:** Go 1.26, charmbracelet/fang, spf13/cobra, spf13/viper, frostyard/std (reporter)

---

### Task 1: Create repository and module scaffold

**Files:**
- Create: `go.mod`
- Create: `Makefile`
- Create: `CLAUDE.md`
- Create: `LICENSE`
- Create: `.gitignore`
- Create: `.svu.yaml`

**Step 1: Create the GitHub repo**

```bash
gh repo create frostyard/clix --public --clone --description "CLI convenience module for Frostyard tools"
cd clix
```

**Step 2: Initialize go module**

```bash
go mod init github.com/frostyard/clix
```

Then edit `go.mod` to set Go version:

```
module github.com/frostyard/clix

go 1.26
```

**Step 3: Create Makefile**

Copy the Makefile from `frostyard/std` — it uses the same `fmt`, `lint`, `test`, `check`, `bump` targets. Identical file, no changes needed.

**Step 4: Create CLAUDE.md**

```markdown
# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`github.com/frostyard/clix` is a CLI convenience module for Frostyard tools. It wraps charmbracelet/fang and spf13/cobra with standardized version injection, common flags, JSON output helpers, and reporter factory.

## Commands

` `` bash
make test            # run all tests
make lint            # run golangci-lint
make check           # fmt + lint + test (pre-commit gate)
make bump            # tag next semver with svu and push
go test -v -run TestName ./...  # run a single test
` ``

## Architecture

Single flat package `clix` with four source files:

- **clix.go** — `App` struct with `Run()` and `VersionString()`. Wires up fang.Execute with version string and signal handling.
- **flags.go** — Package-level flag variables (`JSONOutput`, `Verbose`, `DryRun`), registration on cobra commands, and optional `BindViper()`.
- **output.go** — `OutputJSON()` and `OutputJSONError()` helpers for standardized JSON output to stdout.
- **reporter.go** — `NewReporter()` factory that returns TextReporter or JSONReporter based on `--json` flag.

## Conventions

- Go 1.26; use modern Go syntax (range-over-int, omitzero, etc.)
- One test file per source file, standard `testing` package only
- Tests use fresh `cobra.Command` per test to avoid flag state leakage
- Tests capture output via `bytes.Buffer`; JSON tests unmarshal and validate fields
```

**Step 5: Create LICENSE**

MIT license, 2026, Frostyard. Copy from `frostyard/std/LICENSE`.

**Step 6: Create .gitignore**

```
coverage.out
coverage.html
```

**Step 7: Create .svu.yaml**

Copy from `frostyard/std/.svu.yaml`.

**Step 8: Commit**

```bash
git add -A
git commit -m "feat: initial module scaffold"
```

---

### Task 2: Implement VersionString (TDD)

**Files:**
- Create: `clix.go`
- Create: `clix_test.go`

**Step 1: Write the failing test**

```go
package clix

import "testing"

func TestVersionString(t *testing.T) {
	app := App{
		Version: "1.2.3",
		Commit:  "abc123",
		Date:    "2026-03-04",
		BuiltBy: "ci",
	}
	got := app.VersionString()
	want := "1.2.3 (Commit: abc123) (Date: 2026-03-04) (Built by: ci)"
	if got != want {
		t.Errorf("VersionString() = %q, want %q", got, want)
	}
}

func TestVersionStringDefaults(t *testing.T) {
	app := App{}
	got := app.VersionString()
	want := "dev (Commit: none) (Date: unknown) (Built by: local)"
	if got != want {
		t.Errorf("VersionString() = %q, want %q", got, want)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestVersionString ./...`
Expected: FAIL — `App` type not defined.

**Step 3: Write minimal implementation**

```go
// Package clix provides CLI convenience functions for Frostyard tools,
// wrapping charmbracelet/fang and spf13/cobra with standardized version
// injection, common flags, JSON output helpers, and reporter factory.
package clix

import "fmt"

// App holds build-time metadata for a CLI application.
// Create one in main() and call Run() to execute the root command.
type App struct {
	Version string
	Commit  string
	Date    string
	BuiltBy string
}

// defaults fills zero-value fields with sensible defaults.
func (a *App) defaults() {
	if a.Version == "" {
		a.Version = "dev"
	}
	if a.Commit == "" {
		a.Commit = "none"
	}
	if a.Date == "" {
		a.Date = "unknown"
	}
	if a.BuiltBy == "" {
		a.BuiltBy = "local"
	}
}

// VersionString returns a formatted version string including commit, date,
// and builder info. Example: "1.2.3 (Commit: abc) (Date: 2026-01-01) (Built by: ci)"
func (a *App) VersionString() string {
	a.defaults()
	return fmt.Sprintf("%s (Commit: %s) (Date: %s) (Built by: %s)",
		a.Version, a.Commit, a.Date, a.BuiltBy)
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestVersionString ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add clix.go clix_test.go
git commit -m "feat: add App struct and VersionString"
```

---

### Task 3: Implement common flags (TDD)

**Files:**
- Create: `flags.go`
- Create: `flags_test.go`

**Step 1: Write the failing test**

```go
package clix

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRegisterFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}

	registerFlags(cmd)

	flags := []struct {
		name      string
		shorthand string
	}{
		{"json", ""},
		{"verbose", "v"},
		{"dry-run", "n"},
	}

	for _, f := range flags {
		pf := cmd.PersistentFlags().Lookup(f.name)
		if pf == nil {
			t.Errorf("flag --%s not registered", f.name)
			continue
		}
		if f.shorthand != "" && pf.Shorthand != f.shorthand {
			t.Errorf("flag --%s shorthand = %q, want %q", f.name, pf.Shorthand, f.shorthand)
		}
	}
}

func TestBindViper(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	registerFlags(cmd)

	err := BindViper(cmd)
	if err != nil {
		t.Fatalf("BindViper() error = %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestRegister ./...`
Expected: FAIL — `registerFlags` not defined.

**Step 3: Write minimal implementation**

```go
package clix

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Common flag values, populated when Run() registers persistent flags.
var (
	JSONOutput bool // --json flag value
	Verbose    bool // --verbose / -v flag value
	DryRun     bool // --dry-run / -n flag value
)

// registerFlags adds --json, --verbose, and --dry-run as persistent flags on cmd.
func registerFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&JSONOutput, "json", false, "output in JSON format")
	cmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
	cmd.PersistentFlags().BoolVarP(&DryRun, "dry-run", "n", false, "dry run mode (no actual changes)")
}

// BindViper binds the common flags (--json, --verbose, --dry-run) to viper.
// Call this in a PersistentPreRunE if your app uses viper for config management.
func BindViper(cmd *cobra.Command) error {
	for _, name := range []string{"json", "verbose", "dry-run"} {
		if err := viper.BindPFlag(name, cmd.PersistentFlags().Lookup(name)); err != nil {
			return err
		}
	}
	return nil
}
```

**Step 4: Run `go mod tidy` to fetch dependencies**

Run: `go mod tidy`

**Step 5: Run test to verify it passes**

Run: `go test -v -run "TestRegister|TestBind" ./...`
Expected: PASS

**Step 6: Commit**

```bash
git add flags.go flags_test.go go.mod go.sum
git commit -m "feat: add common flags and BindViper"
```

---

### Task 4: Implement App.Run (TDD)

**Files:**
- Modify: `clix.go`
- Modify: `clix_test.go`

This is the trickiest piece — `fang.Execute` runs the full cobra command loop. We test that `Run()` registers flags and calls fang.Execute by executing a command that captures flag state.

**Step 1: Write the failing test**

Add to `clix_test.go`:

```go
func TestRunRegistersFlags(t *testing.T) {
	// Reset package-level flag state
	JSONOutput = false
	Verbose = false
	DryRun = false

	ran := false
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			ran = true
			return nil
		},
	}

	app := App{Version: "1.0.0"}
	// fang.Execute runs the command; we verify flags were registered
	err := app.Run(cmd)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !ran {
		t.Error("command RunE was not called")
	}

	// Verify flags exist on the command
	if cmd.PersistentFlags().Lookup("json") == nil {
		t.Error("--json flag not registered")
	}
	if cmd.PersistentFlags().Lookup("verbose") == nil {
		t.Error("--verbose flag not registered")
	}
	if cmd.PersistentFlags().Lookup("dry-run") == nil {
		t.Error("--dry-run flag not registered")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestRunRegistersFlags ./...`
Expected: FAIL — `Run` method not defined.

**Step 3: Write minimal implementation**

Add to `clix.go`:

```go
import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

// Run registers common persistent flags on cmd, then executes the command
// via fang.Execute with the formatted version string and signal handling.
func (a *App) Run(cmd *cobra.Command) error {
	a.defaults()
	registerFlags(cmd)
	return fang.Execute(
		context.Background(),
		cmd,
		fang.WithVersion(a.VersionString()),
		fang.WithNotifySignal(os.Interrupt, os.Kill),
	)
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestRunRegistersFlags ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add clix.go clix_test.go
git commit -m "feat: add App.Run with fang execution"
```

---

### Task 5: Implement JSON output helpers (TDD)

**Files:**
- Create: `output.go`
- Create: `output_test.go`

**Step 1: Write the failing tests**

```go
package clix

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"testing"
)

func TestOutputJSON_Active(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	JSONOutput = true
	defer func() { JSONOutput = false }()

	data := map[string]string{"key": "value"}
	ok := OutputJSON(data)

	_ = w.Close()
	os.Stdout = old

	if !ok {
		t.Error("OutputJSON() returned false when JSONOutput is true")
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	var got map[string]string
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if got["key"] != "value" {
		t.Errorf("got key=%q, want %q", got["key"], "value")
	}
}

func TestOutputJSON_Inactive(t *testing.T) {
	JSONOutput = false
	ok := OutputJSON("anything")
	if ok {
		t.Error("OutputJSON() returned true when JSONOutput is false")
	}
}

func TestOutputJSONError(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	JSONOutput = true
	defer func() { JSONOutput = false }()

	err := OutputJSONError("deploy failed", errors.New("timeout"))

	_ = w.Close()
	os.Stdout = old

	if err == nil {
		t.Fatal("OutputJSONError() returned nil error")
	}
	if err.Error() != "deploy failed: timeout" {
		t.Errorf("error = %q, want %q", err.Error(), "deploy failed: timeout")
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if got["error"] != true {
		t.Errorf("error field = %v, want true", got["error"])
	}
	if got["message"] != "deploy failed" {
		t.Errorf("message = %v, want %q", got["message"], "deploy failed")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestOutput ./...`
Expected: FAIL — `OutputJSON` not defined.

**Step 3: Write minimal implementation**

```go
package clix

import (
	"encoding/json"
	"fmt"
	"os"
)

// OutputJSON writes data as indented JSON to stdout if JSONOutput is true.
// Returns true if output was written, false if JSON mode is not active.
func OutputJSON(data any) bool {
	if !JSONOutput {
		return false
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(data)
	return true
}

// OutputJSONError writes a structured error object as JSON to stdout and
// returns a wrapped error for the caller to propagate.
func OutputJSONError(message string, err error) error {
	errOutput := map[string]any{
		"error":   true,
		"message": message,
		"details": err.Error(),
	}
	_ = OutputJSON(errOutput)
	return fmt.Errorf("%s: %w", message, err)
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestOutput ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add output.go output_test.go
git commit -m "feat: add OutputJSON and OutputJSONError helpers"
```

---

### Task 6: Implement reporter factory (TDD)

**Files:**
- Create: `reporter.go`
- Create: `reporter_test.go`

**Step 1: Write the failing tests**

```go
package clix

import (
	"testing"

	"github.com/frostyard/std/reporter"
)

func TestNewReporter_JSON(t *testing.T) {
	JSONOutput = true
	defer func() { JSONOutput = false }()

	r := NewReporter()
	if !r.IsJSON() {
		t.Error("NewReporter() with JSONOutput=true should return JSON reporter")
	}
	if _, ok := r.(*reporter.JSONReporter); !ok {
		t.Errorf("NewReporter() type = %T, want *reporter.JSONReporter", r)
	}
}

func TestNewReporter_Text(t *testing.T) {
	JSONOutput = false

	r := NewReporter()
	if r.IsJSON() {
		t.Error("NewReporter() with JSONOutput=false should return text reporter")
	}
	if _, ok := r.(*reporter.TextReporter); !ok {
		t.Errorf("NewReporter() type = %T, want *reporter.TextReporter", r)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestNewReporter ./...`
Expected: FAIL — `NewReporter` not defined.

**Step 3: Write minimal implementation**

```go
package clix

import (
	"os"

	"github.com/frostyard/std/reporter"
)

// NewReporter returns the appropriate reporter based on the JSONOutput flag.
// JSON mode: JSONReporter writing to os.Stdout (for piping/parsing).
// Text mode: TextReporter writing to os.Stderr (keeps stdout clean for data).
func NewReporter() reporter.Reporter {
	if JSONOutput {
		return reporter.NewJSONReporter(os.Stdout)
	}
	return reporter.NewTextReporter(os.Stderr)
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestNewReporter ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add reporter.go reporter_test.go
git commit -m "feat: add NewReporter factory"
```

---

### Task 7: Full test suite, tidy, and verify

**Files:**
- Modify: `go.mod` (tidy)

**Step 1: Run full test suite**

Run: `make check`
Expected: All formatting, linting, and tests pass.

**Step 2: Verify go.mod is clean**

Run: `go mod tidy && git diff go.mod go.sum`
Expected: No changes (already tidy).

**Step 3: Review test coverage**

Run: `make test-cover`
Review `coverage.html` — all public functions should be covered.

**Step 4: Commit if any tidy changes**

```bash
git add -A
git commit -m "chore: tidy module and verify test coverage"
```

(Skip if nothing changed.)

---

### Task 8: Tag initial release

**Step 1: Verify clean tree**

Run: `git status`
Expected: Clean working directory.

**Step 2: Tag v0.1.0**

```bash
make bump
```

This runs `make check`, then tags with `svu next` and pushes.

---
