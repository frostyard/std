# Spec: problems package

Package `problems` (`github.com/frostyard/std/problems`) writes problem
records (panics and coded errors) of a frostyard app to the local systemd
journal, and reads them back. It implements the writer and reader side of
[core `docs/specs/problem-report.md`](https://github.com/frostyard/core/blob/main/docs/specs/problem-report.md)
(schema `frostyard.problem/1`, kind S8 `app_problem`, the journal record
registry). Consumers: frostyard Go apps (writers), chairlift (reader), and
frostyard Rust apps through the Rust crate `frostyard-problems` in `rust/`
(see [Rust crate](#rust-crate)), which follows the same writer rules and
`Validate`, and runs the same vectors. Neither sends anything over a network.

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
| 7 | `FROSTYARD_RUNTIME` | `go` (the Rust crate writes `rust`) |
| 8 | `FROSTYARD_COMPONENT` | The component |
| 9 | `FROSTYARD_CLASS` | `panic` or `error` |
| 10 | `FROSTYARD_CODE` or `FROSTYARD_FRAMES` | The code (error), or the frames joined by 1 `\n` (panic). The Rust crate writes `FROSTYARD_LOCATION` for a panic instead of frames, and no row 10 for a panic with no location |

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
- Step 1 also removes stale temporary files: for each `*.crash-new` file,
  open it with `O_NOFOLLOW`; when it is at least 1 hour old (mtime) and
  `flock(LOCK_EX|LOCK_NB)` succeeds, remove it with no record. It is never
  read or parsed. A younger file can belong to a live process that has not
  locked it yet, so it stays. At most 8 for each start.
- Step 2, the own file: `os.CreateTemp(dir, "*.crash-new")` (a name that
  step 1 never parses), `flock(LOCK_EX)`, the header line, then publish it:
  `os.Link` to the same name with the `.crash` suffix (a link never replaces
  an existing file; when the name is taken, start again with a new name, at
  most 8 tries), and remove the `.crash-new` name. Only then
  `debug.SetCrashOutput(f, debug.CrashOptions{})`. Thus a concurrent start
  never sees an unlocked `*.crash` file of a live process. The file stays
  open and locked for the life of the process. Only the first successful
  call in a process does this; a later call returns nil.
- Header line: `frostyard-problems/1 app=<appID> version=<version> component=<component>\n`.
- The marker line `frostyard-problems: recorded`: `RecoverPanic` appends it
  to the own file after it sent a record, before it panics again.
- A leftover gives no record when: the header is not valid (4 fields, these
  keys, and an identity that `New` accepts), there is no `panic: ` or
  `fatal error: ` line, the first `goroutine N [...]:` block after it gives
  no frame, or ALL 3 of these are true: the file holds the marker line, the
  first crash message is a panic message whose last line ends with
  ` [recovered, repanicked]`, and no continuation line of that message
  starts with `panic: ` (the re-panic of 1 panic that `RecoverPanic`
  recorded). A `fatal error: ` crash, a panic message without that suffix,
  or a nested panic chain gives 1 record even when the marker is in the
  file. A duplicate record is better than a lost record.
- Panic message: the first `panic: ` line plus the tab-indented
  continuation lines that follow it directly. Go 1.26 prints a multi-line
  panic value that way (`panic: first`, then a tab and
  `second [recovered, repanicked]`), so the suffix is on the last line. Go
  1.26 also prints each nested panic as a continuation line that starts
  with `panic: ` after its tab (`printpanics`: `panic: later failure`, then
  a tab and `panic: cleanup failure [recovered, repanicked]`); such a block
  is a nested panic chain.
- The record of a leftover: the identity of the header (the run that
  crashed), class `panic`, `PRIORITY=2`, `MESSAGE` = the panic message (its
  lines joined by `\n`, the leading tabs removed) or the first
  `fatal error: ` line, cut to 2048 bytes, `FROSTYARD_FRAMES` = the
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
`reject`). The Go tests and the Rust tests read every vector. Each writer
writes only the `write` vectors of its own runtime (`in.runtime`); every
`write` vector with `expect: ok` and every `validate` vector goes through
both validators, and both run every `frame` vector. For a Rust `write`
vector, `in.location` is the raw `file:line` of the panic location and
`in.caller_file` and `in.manifest_dir` the caller file and the
`CARGO_MANIFEST_DIR` of `install_panic_hook`; the writer derives the crate
root from them as the hook does, and maps the location with
`map_location(crate_root, file, line)`.

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
  later crashes, the later crash gives a record when it is a fatal error, a
  plain panic, or a nested panic chain. The remaining limit: a later crash
  that is a single re-panic (` [recovered, repanicked]`, no nested
  `panic: ` line) and whose own record was dropped gives no record, because
  the marker of the earlier panic is in the file.
- A nested panic chain of a panic that `RecoverPanic` recorded (a deferred
  function under the recorder panics again) gives a second record at the
  next start: a duplicate record is better than a lost record.

## Rust crate

The crate `frostyard-problems` (lib `frostyard_problems`, folder `rust/`,
decision [ADR-0003](../adr/0003-rust-crate-frostyard-problems-in-rust-folder.md))
is the Rust writer and validator. It follows the [Writer](#writer) rules,
the [journal fields](#journal-fields-of-a-record) and rules 1 to 7 and 10
of this spec; it has no reader (`Read`, `ParseJSON`) and no crash files. It
uses only the Rust standard library, also in tests (edition 2024, MSRV
1.85, `publish = false`). Apps use a cargo git dependency on a std tag:

```toml
frostyard-problems = { git = "https://github.com/frostyard/std", tag = "vX.Y.Z" }
```

### API

```rust
pub const MESSAGE_ID: &str;      // "8915bdd215c84cbab0c31c05a1a33e7e"
pub const SCHEMA: &str;          // "frostyard.problem/1"
pub const DEFAULT_SOCKET: &str;  // "/run/systemd/journal/socket"

pub enum Component { Daemon, Cli, Gui, Shell, Other }
impl Component { pub const fn as_str(self) -> &'static str; }

#[derive(Clone)] pub struct Recorder; // Send + Sync
impl Recorder {
    pub fn new(app_id: &str, version: &str, component: Component) -> Recorder;
    pub fn with_socket(self, path: impl Into<PathBuf>) -> Recorder;
    pub fn err(&self) -> Option<&str>;
    pub fn error(&self, code: &str, err: Option<&dyn std::fmt::Display>);
    #[track_caller]
    pub fn install_panic_hook(&self, manifest_dir: &str); // env!("CARGO_MANIFEST_DIR")
}

pub fn map_location(crate_root: &str, file: &str, line: u32) -> Option<String>;
pub fn validate(fields: &HashMap<String, String>) -> Result<Detail, Invalid>;
pub fn decode_entry(b: &[u8]) -> Result<HashMap<String, String>, DecodeError>;

pub struct Detail {
    pub app_id: String, pub app_version: String, pub runtime: String,
    pub component: String, pub class: String, pub code: Option<String>,
    pub frames: Vec<String>, pub location: Option<String>,
}
```

| Item | Behavior |
| --- | --- |
| `Recorder::new` | Never fails. Inert when `app_id` is off the allowlist or `version` fails the v1 semver pattern (the component is an enum, so it is always valid) |
| `with_socket` | The journal socket path; default `DEFAULT_SOCKET`. For tests of apps |
| `err` | Why the recorder is inert, or `None` |
| `error` | As Go `Error`: class `error`, `PRIORITY=3`, `FROSTYARD_CODE=code`, `MESSAGE` = the `Display` text of `err` (the code when `err` is `None`, or when `Display` returns an error or panics). A bad code writes nothing. `err` is formatted only when a record can be written (not inert, a valid code, Linux), inside `std::panic::catch_unwind`: a panicking `Display` never unwinds out of `error`. The payload of a caught panic is leaked with `std::mem::forget`, not dropped, because its `Drop` can panic outside the catch boundary. With `panic = "abort"` it still aborts (an installed hook records that panic first). On other OSes `error` returns at once and calls no `Display` |
| `install_panic_hook` | Takes the current hook (`std::panic::take_hook`) and sets a hook that FIRST records the panic, THEN calls the previous hook with the same `PanicHookInfo`. The record: class `panic`, `PRIORITY=2`, `MESSAGE` = the payload when it is a `&str` or a `String`, else `Box<dyn Any>`, and `FROSTYARD_LOCATION` only when `map_location` with the crate root returns a value. `#[track_caller]`: the crate root comes from the file of the caller and `manifest_dir` (`env!("CARGO_MANIFEST_DIR")` of the app; see [Location rule](#location-rule)). The work of the crate in the hook never panics; a previous hook that panics (a panic inside a panic hook aborts) or blocks is outside that guarantee. An inert recorder installs the hook too: it records nothing and calls the previous hook |
| `map_location` | See [Location rule](#location-rule) |
| `validate` | Exactly the rules of Go `Validate` (rule 7), also for Go records: the frame pattern, the 160-byte frame limit and the 32-frame limit are in the crate too |
| `decode_entry` | As Go `DecodeEntry`; also an error for a value that is not UTF-8 |

The field order of a Rust record is the order of the
[journal fields](#journal-fields-of-a-record), with `FROSTYARD_RUNTIME=rust`
and row 10 `FROSTYARD_CODE` (error) or `FROSTYARD_LOCATION` (panic with a
location); a panic with no location has 9 fields. The transport (Linux
only) is 1 `send_to` on a new `UnixDatagram::unbound()` with
`set_nonblocking(true)`; every error drops the record. The rate cap (10
records in any 60 seconds) is a process-wide static. On other OSes a
`Recorder` writes nothing and the hook only calls the previous hook.

### Location rule

The rule is "the crate that installs the hook".

**Crate root.** The app calls
`rec.install_panic_hook(env!("CARGO_MANIFEST_DIR"))` directly in its own
crate. `install_panic_hook(&self, manifest_dir: &str)` is `#[track_caller]`
and reads `std::panic::Location::caller().file()`, the file of the app code
that installs the hook, relative to the workspace root. `manifest_dir` is
the absolute folder of the app crate that cargo gives at compile time.

1. A caller file that is absolute, holds a `\` or a `:`, or has no `src`
   folder component (`examples/x.rs`) gives no root.
2. The candidates are the prefixes of the caller file before each `src`
   folder component: `src/apps/mail/src/main.rs` gives `""` and
   `src/apps/mail/`.
3. A non-empty candidate `R` (it ends with `/`) matches when the path
   components of `R` are the last components of `manifest_dir` (a trailing
   `/` removed): `src-tauri/` matches `/b/app/src-tauri`, `src/apps/mail/`
   matches `/b/ws/src/apps/mail`, `src-tauri/` does not match
   `/b/app/xsrc-tauri`. An empty or relative `manifest_dir` matches nothing.
4. The root is the longest matching non-empty candidate.
5. When no non-empty candidate matches, the root is `""` only when `""` is
   the only candidate (the caller file starts with `src/` and has no other
   `src` folder: `src/main.rs`, `src/bin/x.rs`). Else there is no root
   (unknown or ambiguous).

With no root, no panic of the process gets a location. The root lives in
the hook, not in the `Recorder`.

**`map_location(crate_root, file, line)`** returns `Some("src/<rest>:<line>")`
only when all are true, else `None`:

1. `crate_root` is `""` or ends with `/`.
2. `file` is relative: it does not start with `/`, and holds no `\` and no
   `:`.
3. `file` starts exactly with `crate_root` + `src/`; `<rest>` is what
   follows (`src-tauri/src/sync/imap.rs` with root `src-tauri/` gives
   `src/sync/imap.rs`).
4. The result matches
   `^src/([a-z0-9_]+/){0,8}[a-z0-9_]+\.rs:[1-9][0-9]{0,5}$` and is at most
   128 bytes.

Cargo gives workspace members a path relative to the workspace root, so the
files of the app's crate share the prefix of the file that installs the
hook. crates.io, git and vendored dependencies get an absolute path, and the
standard library `/rustc/<hash>/library/...`; relative paths of other
crates (`vendor/serde/src/...`, `library/core/src/...` after a remapping,
another workspace member) do not start with `<crate_root>src/`. The limits
are in [ADR-0003](../adr/0003-rust-crate-frostyard-problems-in-rust-folder.md).

### Vectors the crate runs

- `write` with `in.runtime: rust`: the writer gets the `in.*` values through
  a test socket (the panic path with the raw `in.location` and
  `in.caller_file` and `in.manifest_dir`), and the
  datagram must hold the `journal.*` fields in order and no other; with
  `expect: reject`, no datagram.
- Every `write` with `expect: ok` (both runtimes) and every `validate`:
  `validate` returns exactly the `payload.*` fields, or rejects.
- Every `frame`: the crate-private frame mapping gives `out.frame`.

`make rust-check` runs `cargo fmt --check`, `cargo clippy --all-targets
--locked -- -D warnings`, `cargo test --locked`, `cargo check --locked` for
`x86_64-pc-windows-gnu` and `x86_64-apple-darwin`, and fails unless `cargo
tree --locked --edges normal,build,dev --prefix none` prints only the crate.

## Derived artifacts

| Artifact | Derivation |
| --- | --- |
| `problems/problems.go` patterns, allowlist and enums | Copied from core `docs/specs/schemas/problem-report/v1.json` |
| `problems/testdata/vectors/v1.txt` | The journal fields and S8 payload fields of the core registry |
| Rust crate `frostyard-problems` (`rust/`) | This spec ([Rust crate](#rust-crate)); its matchers in `rust/src/patterns.rs` are hand-written ports of the patterns of `problems/problems.go` and run the same vectors |

## References

- Rationale: [ADR-0002](../adr/0002-problems-package-writes-to-the-journal-only.md),
  [ADR-0003](../adr/0003-rust-crate-frostyard-problems-in-rust-folder.md) (Rust crate),
  [core ADR-0069](https://github.com/frostyard/core/blob/main/docs/adr/0069-report-problems-local-first-opt-in-with-one-sender-per-host.md)
- Contract: [core spec problem-report.md](https://github.com/frostyard/core/blob/main/docs/specs/problem-report.md)
- Context: [design/overview.md](../design/overview.md)
