# Security policy

std is a Go library (`reporter`, `problems`) with no dependencies, and a Rust
crate (`frostyard-problems` in `rust/`) that uses only the Rust standard
library. Its `problems` code writes problem records to the local systemd
journal and reads them back. Please report security problems privately, as
described below.

## Supported versions

Only the latest release tag receives security fixes. Today that is `v0.5.0`;
see the [releases page](https://github.com/frostyard/std/releases) for the
current one. The Go module and the Rust crate share the same tags: a fix
ships in 1 tag for both. Older tags do not receive fixes — update to the
latest tag.

## Report a vulnerability

Do not open a public issue.

Use GitHub private vulnerability reporting:
https://github.com/frostyard/std/security. Click "Report a vulnerability",
or use the direct form
https://github.com/frostyard/std/security/advisories/new.

If you cannot use GitHub, email the maintainer at
[bketelsen@gmail.com](mailto:bketelsen@gmail.com).

## What happens next

The maintainer confirms receipt of the report. The maintainer then fixes the
problem on `main`, tags a patch release, and publishes a security advisory.

## Scope

The library runs inside the app that imports it, with that app's rights.
These areas are the most sensitive:

- The journal transport: 1 non-blocking `AF_UNIX` datagram per record to
  `/run/systemd/journal/socket` (`problems/transport_linux.go`, `rust/src/transport.rs`).
  A record is dropped silently on any send failure.
- The reader and the record validation (`problems/reader.go`,
  `problems/problems.go`, `rust/src/validate.rs`): the output of
  `journalctl -o json --all` is untrusted input, and only journald's own `_`
  fields are trusted.
- The crash files written by `CrashOutput` (`problems/crash_linux.go`): the
  scanner reads a marked crash file at the next start, so a file planted in
  the crash folder is untrusted input too.
