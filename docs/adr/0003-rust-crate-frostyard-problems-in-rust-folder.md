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
  still run, and the record is in the journal before an abort. The hook
  never panics (a poisoned lock is used as is).
- **"Our crates only" location rule.** `map_location` keeps a location only
  for a relative path with a `src` component, with no
  `<name>-<major>.<minor>.<patch>` folder before it, and only when the result
  `src/...rs:<line>` matches the v1 location pattern (128 bytes at most). The
  result starts at the first `src` component (`src-tauri/src/bridge.rs` gives
  `src/bridge.rs`). Absolute paths (dependencies, the standard library) and
  `trim-paths` paths of dependencies (`serde-1.0.219/src/...`) give no
  location.
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
- The location rule sees only the path. A path dependency inside the
  workspace counts as our crate; a workspace folder named like a crate
  release hides the location; path remapping can let another path through
  when the remapped folder has no version in its name; and a panic outside a
  `src` folder (`build.rs`, `tests/`) has no location.
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
- **Send the full panic path, or the frames of a backtrace:** paths hold
  user names and dependency details; the v1 schema allows only
  `src/...rs:<line>` for Rust.

## References

- Shapes: [specs/problems-package.md](../specs/problems-package.md),
  [design/overview.md](../design/overview.md), the crate's
  [README](../../rust/README.md)
- Builds on: [ADR-0002](0002-problems-package-writes-to-the-journal-only.md),
  [core spec problem-report.md](https://github.com/frostyard/core/blob/main/docs/specs/problem-report.md)
