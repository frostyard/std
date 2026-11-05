# Spec: problems package

Package `problems` (`github.com/frostyard/std/problems`) writes problem
records (panics and coded errors) of a frostyard app to the local systemd
journal, and reads them back. It implements the writer and reader side of
[core `docs/specs/problem-report.md`](https://github.com/frostyard/core/blob/main/docs/specs/problem-report.md)
(schema `frostyard.problem/1`, kind S8 `app_problem`, the journal record
registry). Consumers: frostyard Go apps (writers), chairlift (reader), and
the Rust crate `frostyard-problems` (task t4), which runs the same vectors.
The package never sends anything over a network.

## Interface

### Constants and types

```go
const MessageID = "8915bdd215c84cbab0c31c05a1a33e7e"
const Schema = "frostyard.problem/1"
const DefaultSocket = "/run/systemd/journal/socket"

type Component string
const (
    ComponentDaemon Component = "daemon"
    ComponentCLI    Component = "cli"
    ComponentGUI    Component = "gui"
    ComponentShell  Component = "shell"
    ComponentOther  Component = "other"
)

var ErrInvalid error // wrapped by every Validate error
```

### Writer

```go
func New(appID, version string, component Component, opts ...Option) *Recorder
func WithSocket(path string) Option
func (r *Recorder) Err() error
func (r *Recorder) Error(code string, err error)
func (r *Recorder) RecoverPanic()
func (r *Recorder) CrashOutput() error
func MapFrame(name string) string
func DecodeEntry(b []byte) (map[string]string, error)
```

| Call | Behavior |
| --- | --- |
| `New` | Never fails. The recorder is inert (writes nothing) when `appID` is not in the S8 allowlist, `version` does not match the spec `<semver>`, or `component` is not in the S8 enum. Options apply first |
| `WithSocket` | The journal socket path; default `DefaultSocket`. For tests of apps |
| `Err` | Why the recorder is inert, or `nil`. A nil `*Recorder` returns an error |
| `Error` | Class `error`, `PRIORITY=3`, `FROSTYARD_CODE=code`, `MESSAGE=err.Error()` (the code when `err` is nil). A code that fails `^[a-z][a-z0-9_.]{0,63}$` writes nothing |
| `RecoverPanic` | Only as `defer r.RecoverPanic()`. With no panic, it does nothing. Else it records class `panic`, `PRIORITY=2`, `MESSAGE` = the panic value as text, `FROSTYARD_FRAMES` = the stack from `runtime.Callers` (without `runtime.Callers` and `RecoverPanic`), innermost first, at most 32, each through `MapFrame`. Then it calls `panic` with the same value: Go prints `[recovered, repanicked]` and exits with code 2 |
| `CrashOutput` | See [Crash files](#crash-files). On an inert recorder it returns `Err()` and does nothing |
| `MapFrame` | `name` when it matches the v1 frame pattern and is at most 160 bytes; else `other` |
| `DecodeEntry` | Decodes 1 native-protocol datagram (binary and `NAME=value\n` forms). It rejects an empty entry, a repeated field, a bad length, a missing final newline, and a name that is not `^[A-Z0-9_]+$` or starts with `_` |

`MESSAGE` is cut to 2048 bytes on a UTF-8 boundary (at most 3 bytes back; a
longer run of continuation bytes is not UTF-8 and is cut at 2048). All
methods are safe on a nil `*Recorder` (inert); `RecoverPanic` on a nil
recorder still panics again. A `Recorder` is safe for concurrent use.

### Journal fields of a record

In this order, and no others:

| # | Field | Value |
| --- | --- | --- |
| 1 | `MESSAGE_ID` | `MessageID` |
| 2 | `PRIORITY` | `2` (panic) or `3` (error) |
| 3 | `MESSAGE` | Error text or panic message (local only) |
| 4 | `FROSTYARD_SCHEMA` | `frostyard.problem/1` |
| 5 | `FROSTYARD_APP_ID` | The app ID |
| 6 | `FROSTYARD_APP_VERSION` | The version |
| 7 | `FROSTYARD_RUNTIME` | `go` |
| 8 | `FROSTYARD_COMPONENT` | The component |
| 9 | `FROSTYARD_CLASS` | `panic` or `error` |
| 10 | `FROSTYARD_CODE` or `FROSTYARD_FRAMES` | The code (error), or the frames joined by 1 `\n` (panic) |

Each field uses the binary form of the journal native protocol: `NAME\n`,
the value length as a little-endian uint64, the value, `\n`.

### Reader

```go
func Read(ctx context.Context, opts ReadOptions) (Result, error)
func ParseJSON(r io.Reader) (Result, error)
func Validate(fields map[string]string) (Detail, error)

type ReadOptions struct {
    User       bool      // --user; else --system
    Since      time.Time // --since=@<unix seconds> when not zero
    Limit      int       // -n <Limit> when above 0
    Journalctl string    // default "journalctl"
}
type Result struct {
    Records []Record
    Dropped int
}
type Record struct {
    Trusted Trusted
    Detail  Detail
    Message string // MESSAGE, local only
}
type Trusted struct {
    UID, Comm, SystemdUnit, SystemdUserUnit, Cursor string
    Time time.Time // __REALTIME_TIMESTAMP, UTC
}
type Detail struct {
    AppID      string   `json:"app_id"`
    AppVersion string   `json:"app_version"`
    Runtime    string   `json:"runtime"`
    Component  string   `json:"component"`
    Class      string   `json:"class"`
    Code       string   `json:"code,omitempty"`
    Frames     []string `json:"frames,omitempty"`
    Location   string   `json:"location,omitempty"`
}
```

`Read` runs:

```text
journalctl --no-pager -q -o json --all --user|--system [--since=@<unix>] [-n <limit>] MESSAGE_ID=8915bdd215c84cbab0c31c05a1a33e7e
```

and gives its standard output to `ParseJSON`. On a non-zero exit it returns
an error with the exit status and at most 1024 bytes of stderr.

### Crash files

- Folder: `$STATE_DIRECTORY` (the first entry when it holds `:`), else
  `$XDG_STATE_HOME`, else `$HOME/.local/state` (only absolute values
  count), then `frostyard-problems/<appID>`. Folders are made with mode
  0700, files have mode 0600. No usable variable: `CrashOutput` returns an
  error.
- Step 1, leftovers: for each `*.crash` file, sorted by name, open it with
  `O_NOFOLLOW` and try `flock(LOCK_EX|LOCK_NB)`. A lock that fails means a
  live process owns the file: skip it. Else read at most 64 KiB, turn it
  into at most 1 record, and remove the file. At most 8 files are handled
  for each start (skipped files do not count).
- Step 2, the own file: `os.CreateTemp(dir, "*.crash")`, `flock(LOCK_EX)`,
  kept open for the life of the process, then the header line, then
  `debug.SetCrashOutput(f, debug.CrashOptions{})`. Only the first successful
  call in a process does this; a later call returns nil.
- Header line: `frostyard-problems/1 app=<appID> version=<version> component=<component>\n`.
- The marker line `frostyard-problems: recorded`: `RecoverPanic` appends it
  to the own file after it sent a record, before it panics again.
- A leftover gives no record when: the header is not valid (4 fields, these
  keys, and an identity that `New` accepts), there is no `panic: ` or
  `fatal error: ` line, the first `goroutine N [...]:` block after it gives
  no frame, or the file holds the marker line AND the first crash line is a
  `panic: ` line that ends with ` [recovered, repanicked]` (the re-panic of a
  panic that `RecoverPanic` recorded). A `fatal error: ` crash, or a
  `panic: ` line without that suffix, gives 1 record even when the marker is
  in the file.
- The record of a leftover: the identity of the header (the run that
  crashed), class `panic`, `PRIORITY=2`, `MESSAGE` = the first `panic: ` or
  `fatal error: ` line (cut to 2048 bytes), `FROSTYARD_FRAMES` = the
  function lines of the first goroutine block (not the tab-indented file
  lines, not `created by` lines, not `...` lines), with the argument list
  `(...)` at the end removed, through `MapFrame`, at most 32.

### Test vectors

`problems/testdata/vectors/v1.txt`, in the format of
[its README](../../problems/testdata/vectors/README.md): blocks separated
by 1 blank line, `#` comment lines, `key: value` lines split at the first
`: `, repeatable keys. Keys: `vector`, `type` (`write`, `validate`,
`frame`), `in.*`, `journal.<FIELD>` (repeated `journal.FROSTYARD_FRAMES`
lines join with `\n`), `payload.<field>`, `out.frame`, `expect` (`ok` or
`reject`). The Go tests read and run every vector; the Rust crate must too.

## Rules

1. The package imports only the Go standard library, also in tests.
2. The package has no network code; it writes only to the journal socket.
3. 1 record is 1 `Sendto` of 1 datagram on a new non-blocking
   `AF_UNIX`/`SOCK_DGRAM` socket (`SOCK_NONBLOCK|SOCK_CLOEXEC`).
   `EMSGSIZE`, `EAGAIN`, `ENOBUFS`, `ENOENT`, `ECONNREFUSED` and every other
   error drop the record: no retry, no wait, no memfd, no error to the
   caller, no panic, no output.
4. At most 10 records in any 60-second window for each process, shared by
   all recorders. A dropped record past the cap is not retried.
5. An inert recorder writes nothing; a bad code writes nothing.
6. No frame holds a file path, a line number, an offset or an argument: a
   frame is a runtime function name through `MapFrame`, or `other`.
7. `Validate` is the only validation path. It rejects: a `MESSAGE_ID` other
   than `MessageID`; a `FROSTYARD_SCHEMA` other than `frostyard.problem/1`;
   a `FROSTYARD_*` field that the registry does not name; an app ID off the
   allowlist; a version, code, frame or location that fails its v1 pattern
   or length; a runtime, component or class outside its enum (no mapping to
   `other`); `class=error` without a code, or with frames or a location;
   `class=panic` with a code; `runtime=go` with a location; `runtime=rust`
   with frames; an empty `FROSTYARD_FRAMES`, or more than 32 frames. It does
   not check `MESSAGE` or fields that do not start with `FROSTYARD_`.
8. `ParseJSON` reads 1 JSON object for each line and skips empty lines. It
   drops and counts an entry that is not a JSON object, that has a field
   given as an array of strings (a repeated field), a `null`, a number or
   any value that is not a string or an array of byte numbers 0 to 255, an
   unparsable `__REALTIME_TIMESTAMP`, or that fails `Validate`. An array of
   byte numbers is decoded to bytes before validation. It returns an error
   only when it cannot read its input.
9. `Read` passes `--all`, so that journalctl prints a field longer than
   4096 bytes (`FROSTYARD_FRAMES` can be up to 5151 bytes) instead of
   `null`.
10. On an OS other than Linux, `New`, `Error` and `CrashOutput` compile and
    do nothing (`CrashOutput` returns `Err()`), `RecoverPanic` records
    nothing and still panics again, and `Read` returns an error that wraps
    `errors.ErrUnsupported`. `ParseJSON`, `Validate`, `MapFrame` and
    `DecodeEntry` work on every OS.

### Limits

- `RecoverPanic` covers only its own goroutine. It must be deferred directly
  (`defer r.RecoverPanic()`); in a wrapper function, `recover` returns nil.
- `RecoverPanic` cannot catch a fatal error (concurrent map writes, out of
  memory, a deadlock) or `os.Exit`.
- `CrashOutput` covers panics of every goroutine and fatal errors, but only
  at the next start of the app, and only for the 64 KiB at the start of the
  crash file.
- When other code recovers the re-panic of `RecoverPanic` and the process
  later crashes, the later crash gives a record when it is a fatal error or a
  plain panic. The remaining limit: a later crash that is also a re-panic
  (` [recovered, repanicked]`) and whose own record was dropped gives no
  record, because the marker of the earlier panic is in the file.

## Derived artifacts

| Artifact | Derivation |
| --- | --- |
| `problems/problems.go` patterns, allowlist and enums | Copied from core `docs/specs/schemas/problem-report/v1.json` |
| `problems/testdata/vectors/v1.txt` | The journal fields and S8 payload fields of the core registry |
| Rust crate `frostyard-problems` (`rust/`, task t4) | This spec and the same vectors |

## References

- Rationale: [ADR-0002](../adr/0002-problems-package-writes-to-the-journal-only.md),
  [core ADR-0069](https://github.com/frostyard/core/blob/main/docs/adr/0069-report-problems-local-first-opt-in-with-one-sender-per-host.md)
- Contract: [core spec problem-report.md](https://github.com/frostyard/core/blob/main/docs/specs/problem-report.md)
- Context: [design/overview.md](../design/overview.md)
