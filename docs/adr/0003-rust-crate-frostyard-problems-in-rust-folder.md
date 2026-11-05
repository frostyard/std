# 0003 — The Rust crate frostyard-problems lives in rust/ of std

- **Status:** Accepted
- **Date:** 2026-11-05

## Context

[ADR-0002](0002-problems-package-writes-to-the-journal-only.md) gave std the
Go package `problems`, the writer and reader of the problem records of
[core `docs/specs/problem-report.md`](https://github.com/frostyard/core/blob/main/docs/specs/problem-report.md)
(schema `frostyard.problem/1`, kind S8 `app_problem`). It also said that
frostyard apps written in Rust (for example Tauri apps) get a Rust crate
that follows the same spec and runs the same shared vectors in
`problems/testdata/vectors/`.

The facts that constrain the crate:

- The Go and the Rust writer must write the same journal fields, and the 2
  validators must accept and reject the same records. The vectors are the
  shared proof; a second repository would have to copy them and could drift.
- Rust has no `recover`: a panic runs the panic hook of the process, then
  unwinds or aborts. Apps and libraries (for example Tauri, or a logging
  crate) set their own hooks too.
- A panic location is a file and a line. Cargo gives rustc the path of a
  workspace member relative to the workspace root, but crates.io, git and
  vendored dependencies (a Flatpak build vendors every crate) get an absolute
  path, and the standard library gets `/rustc/<hash>/library/...`. A path of
  the build host or of a dependency must not reach a record.
- Rust apps run on Linux in the fleet, but their developers also build on
  macOS and Windows.
- std is a Go module with no toolchain but Go; the release is a tag and a
  changelog only.

## Decision

The crate `frostyard-problems` (lib `frostyard_problems`) lives in the
folder `rust/` of std (Brian, plan T857 ask T857d). Its contract is the
section "Rust crate" of [specs/problems-package.md](../specs/problems-package.md).

- **Stdlib only.** The crate has no dependency of any kind, also in tests:
  the v1 patterns are hand-written matchers with exactly the results of the
  Go regular expressions, the transport is
  `std::os::unix::net::UnixDatagram`, and the tests read the vectors with a
  small parser. `make rust-check` fails unless `cargo tree` prints only the
  crate. It has no network code; a test fails on the names of the network
  types in its sources.
- **Same writer rules as Go.** 1 record is 1 non-blocking datagram, every
  error drops the record, at most 10 records in any 60 seconds for each
  process, an inert recorder on a bad identity, and nothing for a bad code.
  `FROSTYARD_RUNTIME` is `rust`; a panic has a `FROSTYARD_LOCATION` instead
  of frames.
- **The panic hook records, then calls the previous hook.**
  `Recorder::install_panic_hook` takes the current hook and sets a hook that
  first writes the record and then calls the previous hook with the same
  `PanicHookInfo`. The usual panic message and the hooks of other libraries
  still run, and the record is in the journal before an abort. The work of
  the crate in the hook never panics (no unwrap, no failing index, a
  poisoned lock is used as is). That guarantee does not cover the previous
  hook: a previous hook that panics aborts the process (a panic inside a
  panic hook aborts), and a previous hook that blocks blocks the panicking
  thread.
- **A panicking `Display` does not unwind out of `error`.** `error` formats
  the error text only when it can write a record (on Linux, after the
  identity and code checks), inside `std::panic::catch_unwind`; a `Display`
  impl that fails or panics gives the code as `MESSAGE`. With
  `panic = "abort"` such a panic still aborts (the installed hook records
  it first).
- **"The crate that installs the hook" location rule.**
  `install_panic_hook` is `#[track_caller]`: it reads the file of its caller
  (`src/main.rs`, `src-tauri/src/main.rs`) and keeps the relative prefix
  before the first `src` folder as the crate root (`""`, `src-tauri/`). A
  caller file that is absolute, holds a `\` or a `:`, or has no `src`
  folder gives no root, and then no panic of the process gets a location.
  `map_location(crate_root, file, line)` keeps a location only when `file`
  is relative and starts exactly with `<crate_root>src/`, and the result
  `src/<rest>:<line>` matches the v1 location pattern (128 bytes at most):
  `src-tauri/src/bridge.rs` with root `src-tauri/` gives `src/bridge.rs`.
  Every other path gives no location: absolute paths (crates.io, git and
  vendored dependencies, the standard library), and relative paths of other
  crates (`vendor/serde/src/...`, `library/core/src/...` after a path
  remapping, `serde-1.0.219/src/...` from `trim-paths`, another workspace
  member). The rule names the app's crate exactly, so it needs no guess
  about the names of dependency folders.
- **A no-op on other OSes.** On every OS but Linux the crate compiles, a
  recorder writes nothing, and the hook only calls the previous hook;
  `validate`, `map_location` and `decode_entry` work everywhere. CI runs a
  check build for `x86_64-pc-windows-gnu` and `x86_64-apple-darwin`.
- **Rust is pinned in `mise.toml`.** rust 1.99.1 with clippy, rustfmt and
  the 2 targets (core ADR-0043). `mise.lock` records the version and
  options only: the mise rust backend installs through rustup, which checks
  each download against the sha256 in the Rust channel manifest. `make
  rust-check` runs cargo fmt, clippy, the tests, and the 2 check builds; it
  is the last step of `make ci` and the CI job `rust` (Rust crate).
- **Consumers use a cargo git dependency on a std tag.** The manifest says
  `publish = false`; an app writes
  `frostyard-problems = { git = "https://github.com/frostyard/std", tag = "vX.Y.Z" }`.
  A std release tag is the release of the crate; the Go release flow does
  not change.

## Consequences

- A change of the contract changes the Go package, the Rust crate and the
  vectors in 1 pull request, and both test suites run every vector.
- std now needs a Rust toolchain for `make ci` (from `mise install`); `make
  check` and `make verify` stay Go only.
- A cargo git dependency fetches the whole std repository, and the crate
  version in `Cargo.toml` is not the release version: the tag is. Dependabot
  of an app sees the tag, not a crates.io version.
- Only the crate that installs the hook gets locations. A panic in another
  crate of the same workspace (a library crate of the app) has no location,
  and neither has a panic in a file of the app's crate outside its `src`
  folder (`build.rs`, `tests/`). A path remapping (`--remap-path-prefix`,
  `trim-paths`) that makes the app's own paths absolute or changes their
  prefix gives no location for any panic. An app must call
  `install_panic_hook` directly in its own crate; called through a helper in
  another crate, it takes the root of that crate.
- The hook covers every thread, but a library that sets its own hook later
  without calling the previous hook removes it. A panic inside a panic hook
  aborts with no record, and a fatal signal (for example a crash in C code)
  gives no record; the crate has no crash file like Go's `CrashOutput`.
- The Rust crate has no reader (`Read`, `ParseJSON`): chairlift is a Go app.

## Alternatives considered

- **A repository of its own (frostyard/problems-rs):** the vectors and the
  contract would live in 2 places and drift; 1 pull request could not change
  both writers.
- **Publish on crates.io:** a public name and a second release flow for a
  crate that only frostyard apps use; a git tag is enough.
- **Use `regex`, `libc` or `serde`:** dependencies for 4 small patterns, 1
  socket call and a line format; the stdlib-only rule of std stays.
- **Call the previous hook first:** the default hook prints first, and with
  `panic = "abort"` a slow output could cost the record; recording first
  costs only 1 non-blocking send.
- **A path heuristic for "our crates" (a `src` folder and no
  `<name>-<x.y.z>` folder before it):** the first version of this crate. A
  cross-review found that it let `vendor/serde/src/...` and
  `library/core/src/...` through; a rule that names the app's crate root
  exactly fails closed instead.
- **Send the full panic path, or the frames of a backtrace:** paths hold
  user names and dependency details; the v1 schema allows only
  `src/...rs:<line>` for Rust.

## References

- Shapes: [specs/problems-package.md](../specs/problems-package.md),
  [design/overview.md](../design/overview.md), the crate's
  [README](../../rust/README.md)
- Builds on: [ADR-0002](0002-problems-package-writes-to-the-journal-only.md),
  [core spec problem-report.md](https://github.com/frostyard/core/blob/main/docs/specs/problem-report.md)
