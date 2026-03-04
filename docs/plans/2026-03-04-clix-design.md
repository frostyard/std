# frostyard/clix Design

## Problem

Three Frostyard CLI tools (nbc, updex, intuneme) all use charmbracelet/fang wrapping spf13/cobra. Each duplicates:

- Version info injection via ldflags (4 variables, identical format string)
- fang.Execute() boilerplate with signal handling
- Common persistent flags (--json, --verbose, --dry-run)
- JSON output helpers (pretty-print to stdout, structured errors)
- Conditional reporter construction based on --json flag

This duplication means bugs and style drift across repos.

## Solution

A new module `github.com/frostyard/clix` — a single flat Go package providing CLI convenience functions built on fang, cobra, and frostyard/std.

Separate from frostyard/std because std is stdlib-only. clix has external dependencies.

## Dependencies

- `github.com/charmbracelet/fang` — CLI execution wrapper
- `github.com/spf13/cobra` — command framework
- `github.com/spf13/viper` — optional flag binding
- `github.com/frostyard/std` — reporter package

Go version: 1.26.

## API

### App struct

```go
type App struct {
    Version string // set via ldflags, default "dev"
    Commit  string // default "none"
    Date    string // default "unknown"
    BuiltBy string // default "local"
}
```

### Core functions

```go
// Run registers common persistent flags (--json, --verbose, --dry-run)
// on cmd, then calls fang.Execute with the formatted version string
// and signal handling (os.Interrupt, os.Kill).
func (a *App) Run(cmd *cobra.Command) error

// VersionString returns "VERSION (Commit: X) (Date: Y) (Built by: Z)"
func (a *App) VersionString() string
```

### Flag variables

```go
var (
    JSONOutput bool // --json flag value
    Verbose    bool // --verbose / -v flag value
    DryRun     bool // --dry-run / -n flag value
)
```

Populated by `Run()` registering persistent flags on the root command. Accessed directly as package-level variables from any command handler.

### Viper integration

```go
// BindViper binds --json, --verbose, and --dry-run to viper.
// Call after Run() if your app uses viper for config management.
func BindViper(cmd *cobra.Command) error
```

### JSON output

```go
// OutputJSON writes data as indented JSON to stdout if --json is active.
// Returns true if output was written, false if --json is not set.
func OutputJSON(data any) bool

// OutputJSONError writes a structured error JSON object to stdout and
// returns the error wrapped with the message for the caller to propagate.
func OutputJSONError(message string, err error) error
```

### Reporter factory

```go
// NewReporter returns a reporter based on the --json flag.
// JSON mode: JSONReporter writing to os.Stdout.
// Text mode: TextReporter writing to os.Stderr.
func NewReporter() reporter.Reporter
```

## File layout

```
github.com/frostyard/clix/
├── clix.go          # App struct, Run(), VersionString()
├── clix_test.go
├── flags.go         # Flag variables, registration, BindViper()
├── flags_test.go
├── output.go        # OutputJSON(), OutputJSONError()
├── output_test.go
├── reporter.go      # NewReporter()
├── reporter_test.go
├── go.mod
├── go.sum
├── Makefile
├── CLAUDE.md
├── LICENSE          # MIT
└── README.md
```

## Consumer usage

### main.go (e.g., nbc)

```go
package main

import (
    "fmt"
    "os"
    "github.com/frostyard/clix"
    "github.com/frostyard/nbc/cmd"
)

var version, commit, date, builtBy = "dev", "none", "unknown", "local"

func main() {
    app := clix.App{
        Version: version, Commit: commit, Date: date, BuiltBy: builtBy,
    }
    if err := app.Run(cmd.RootCmd()); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

### Command handler

```go
func runList(cmd *cobra.Command, args []string) error {
    if clix.OutputJSON(disks) {
        return nil
    }
    // human-readable text output
    for _, d := range disks {
        fmt.Println(d.Name)
    }
    return nil
}
```

### Reporter usage

```go
func runInstall(cmd *cobra.Command, args []string) error {
    progress := clix.NewReporter()
    progress.Step(1, 3, "Preparing disk")
    // ...
    progress.Complete("Installation successful", nil)
    return nil
}
```

### With viper

```go
func init() {
    // After app.Run() registers flags, bind them to viper
    rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
        return clix.BindViper(cmd)
    }
}
```

## Testing strategy

- `bytes.Buffer` for output capture
- Fresh `cobra.Command` per test to avoid flag state leakage
- JSON output validated by unmarshaling
- Reporter factory tested by type assertion
- stdlib `testing` only — no external test libraries

## What this replaces per repo

| Repo | Lines removed (approx) | What goes away |
|------|----------------------|----------------|
| nbc | ~50 | main.go version boilerplate, root.go flag setup, outputJSON/outputJSONError helpers |
| updex | ~40 | main.go version boilerplate, common/common.go flag registration, OutputJSON helper |
| intuneme | ~20 | main.go version + fang boilerplate, makeVersionString |

## Decisions

1. **Separate module, not in std** — std stays stdlib-only.
2. **Single flat package** — simple imports, mirrors std/reporter pattern.
3. **Config struct, not functional options** — idiomatic Go, discoverable, extensible.
4. **Package-level flag variables** — matches updex pattern, simple direct access.
5. **Viper is opt-in** — BindViper() call, not automatic. Keeps the dependency but doesn't force it.
6. **Three common flags** — --json, --verbose/-v, --dry-run/-n. Broad enough to be useful, narrow enough to be universal.
7. **NewReporter() uses os.Stdout/os.Stderr** — JSON to stdout (for piping), text to stderr (keeps stdout clean for data). Matches existing patterns in nbc and updex.
