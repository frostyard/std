<!-- readme-kind: library -->
# std

[![CI](https://github.com/frostyard/std/actions/workflows/ci.yml/badge.svg)](https://github.com/frostyard/std/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/frostyard/std.svg)](https://pkg.go.dev/github.com/frostyard/std)

std is a Go module of standard-library-only shared packages for frostyard
programs — updex, clix, chairlift and other tools — and it carries the Rust
crate `frostyard-problems` for frostyard apps written in Rust.

**Status:** pre-release. Releases are `v0` tags, and the API can change between minor releases.

**Documentation:** [pkg.go.dev/github.com/frostyard/std](https://pkg.go.dev/github.com/frostyard/std)

## What it does

| Package | What it gives you |
| --- | --- |
| [`reporter`](docs/specs/reporter-package.md) | A `Reporter` interface for progress output, with text, JSON Lines and silent implementations. |
| [`problems`](docs/specs/problems-package.md) | Writes the panics and coded errors of an app to the local systemd journal as problem records, and reads them back. |
| [`rust/`](rust/README.md) (`frostyard-problems`) | The same record writer for Rust apps, in the Rust standard library only. |

## Install

```sh
go get github.com/frostyard/std
```

The Rust crate is not on crates.io: add it as a cargo git dependency on a
std tag, as [rust/README.md](rust/README.md) shows.

## Use

```go
package main

import (
	"os"

	"github.com/frostyard/std/reporter"
)

func main() {
	r := reporter.NewTextReporter(os.Stdout)

	r.Step(1, 2, "Download the image")
	r.Progress(50, "12 of 24 MiB")
	r.Step(2, 2, "Write the system extension links")
	r.Complete("Done", nil)
}
```

## Develop

```sh
mise install   # golangci-lint, svu, Rust
make ci        # the canonical gate (what CI's Verify job runs)
```

Design and invariants: [docs/design/overview.md](docs/design/overview.md).

## Where to go next

- [AGENTS.md](AGENTS.md): the rules for agents and contributors.
- [docs/README.md](docs/README.md): the decisions, designs, specs and plans.
- [SECURITY.md](SECURITY.md): report a vulnerability privately.
- [LICENSE](LICENSE): MIT.
