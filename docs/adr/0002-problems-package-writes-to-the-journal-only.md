# 0002 — The problems package writes to the local journal only

- **Status:** Accepted
- **Date:** 2026-11-04

## Context

frostyard decided to report app problems local first, opt in, with 1 sender
for each host
([core ADR-0069](https://github.com/frostyard/core/blob/main/docs/adr/0069-report-problems-local-first-opt-in-with-one-sender-per-host.md)).
The contract is
[core `docs/specs/problem-report.md`](https://github.com/frostyard/core/blob/main/docs/specs/problem-report.md):
schema `frostyard.problem/1`, kind S8 `app_problem`, and the journal record
registry (a fixed `MESSAGE_ID`, the `FROSTYARD_*` fields, closed and open
enums, and portable patterns). Each frostyard app must write a record when it
panics or meets a coded error, and chairlift and `snosi-health` must read the
records back and send only what the user agreed to send.

The facts that constrain the writer and the reader:

- std is stdlib only (a hard rule of this module); updex and clix import it.
- Apps are written in Go now and in Rust later; both must write the same
  records and validate them the same way.
- Anyone on the host can write `FROSTYARD_*` fields to the journal with
  `systemd-cat` (core spec rule 4). Only the fields that journald adds
  itself (`_UID`, `_COMM`, `_SYSTEMD_UNIT`, `_SYSTEMD_USER_UNIT`) are trusted.
- A crashing app cannot wait for the journal, and a record that does not fit
  must not block it (core spec rule 10: 1 record is 1 datagram).
- `recover` works only in the goroutine that panics, only in a function that
  the `defer` statement calls directly, and never for a fatal error
  (concurrent map writes, out of memory, a deadlock) or `os.Exit`.

## Decision

std gets a package `github.com/frostyard/std/problems`, with the spec
[problems-package.md](../specs/problems-package.md):

- **Journal only, never send.** The package writes to the local journal and
  reads it with `journalctl`. It has no network code. The sender is chairlift
  or `snosi-health`, never the app.
- **Go now, Rust later.** The Go package is the first writer and reader. The
  Rust crate `frostyard-problems` (in `rust/`, task t4 of the plan) follows
  the same spec, and both run the shared vectors in
  `problems/testdata/vectors/` (a plain line format that needs no serde).
- **Stdlib only.** The transport is package `syscall` (an `AF_UNIX`
  `SOCK_DGRAM` socket, `SOCK_NONBLOCK|SOCK_CLOEXEC`), the encoding is the
  journal native protocol, and the reader runs `journalctl -o json`. No
  libsystemd, no cgo, no dependency.
- **1 datagram, drop on overflow.** 1 `Sendto` for each record. `EMSGSIZE`,
  `EAGAIN`, `ENOBUFS`, `ENOENT` and `ECONNREFUSED` drop the record: no
  retry, no wait, no memfd fallback, no error to the caller, no output.
- **Rate cap.** At most 10 records in any 60 seconds for each process, shared
  by all recorders, so a loop of errors cannot flood the journal.
- **Inert on a bad identity.** `New` never fails. An app ID off the
  allowlist, a version that is not a spec `<semver>`, or an unknown
  component makes the recorder inert: it writes nothing, and `Err` says why.
  A bad code writes nothing for that call.
- **The reader trusts only journald's `_` fields.** Every `FROSTYARD_*`
  value goes through 1 validation path, `Validate`; a record that fails is
  dropped and counted. `MESSAGE` stays on the host.
- **`RecoverPanic` and `CrashOutput` share the work.** `defer
  r.RecoverPanic()` records a panic of its own goroutine with its frames and
  panics again with the same value. It covers only its own goroutine, it
  must be deferred directly, and it cannot catch a fatal error or `os.Exit`.
  `CrashOutput` covers the rest through `runtime/debug.SetCrashOutput`: a
  locked crash file for each process, turned into 1 record at the next start
  only. The file gets its scanned `*.crash` name only after its lock is held
  (it starts as `*.crash-new`), so a concurrent start cannot remove the file
  of a live process. A marker line in the crash file stops a second record
  for a panic that `RecoverPanic` already recorded: a leftover gives no
  record only when it holds the marker and the last line of its first panic
  message (Go prints a multi-line panic value as tab-indented continuation
  lines) ends with ` [recovered, repanicked]`.

## Consequences

- An app gets problem records with 3 lines: `New`, `defer
  r.RecoverPanic()` in `main` and in each goroutine it starts, and
  `CrashOutput` early in `main`.
- A record can be lost: when the journal is busy, when the record is too
  big, over the rate cap, or when there is no journal (a container). This is
  deliberate: the app never waits for diagnostics.
- A crash in another goroutine, or a fatal error, is recorded only when the
  app starts again, with the version of the run that crashed. An app that
  never starts again leaves its crash file.
- Every cleanly ended run leaves a crash file with only its header. The next
  start removes it (at most 8 files for each start).
- When other code recovers the re-panic of `RecoverPanic` (for example
  `net/http` for a handler) and the process crashes later, the later crash
  still gives 1 record when it is a fatal error or a plain panic. The
  remaining limit: a later crash that is also a re-panic (its first `panic: `
  line ends with ` [recovered, repanicked]`) and whose own record was dropped
  (busy journal, rate cap, no journal) gives no record, because the marker of
  the earlier panic is in the file. `RecoverPanic` is for the top of a
  goroutine, not for code under such a recover.
- A change to the contract is a core spec change first, then a change to
  `problems.go`, the vectors, and the Rust crate together.
- The reader needs `journalctl` on `PATH` (or a path in `ReadOptions`), and
  `--all`, because `FROSTYARD_FRAMES` can be longer than the 4096 bytes that
  `journalctl -o json` prints without it.

## Alternatives considered

- **Use go-systemd or libsystemd (`sd_journal_send`):** breaks the stdlib
  only rule, and libsystemd needs cgo.
- **Write records to a file or send them from the app:** a second store
  beside the journal, or a network sender in every app, which core ADR-0069
  rejects.
- **Block or retry on a full journal socket, or fall back to a memfd for a
  big record:** a crashing app could hang, and core spec rule 10 says a
  record that does not fit is dropped.
- **Only `RecoverPanic`:** misses panics of other goroutines and every fatal
  error, which are the crashes that matter most.
- **Map unknown values to `other` in the reader:** a journal record is
  untrusted; in schema v1 an unknown value is invalid, so the reader drops
  the record (only a writer maps a frame to `other`).

## References

- Shapes: [specs/problems-package.md](../specs/problems-package.md),
  [design/overview.md](../design/overview.md)
- Builds on: [core ADR-0069](https://github.com/frostyard/core/blob/main/docs/adr/0069-report-problems-local-first-opt-in-with-one-sender-per-host.md),
  [core spec problem-report.md](https://github.com/frostyard/core/blob/main/docs/specs/problem-report.md)
