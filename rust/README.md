# frostyard-problems

The Rust crate of the frostyard problem records: it writes the panics and
coded errors of a frostyard Rust app to the local systemd journal, as the Go
package [`problems`](../problems/) does. It never sends anything over a
network, and it uses only the Rust standard library.

- Contract: [docs/specs/problems-package.md](../docs/specs/problems-package.md),
  section "Rust crate"
- Decision: [ADR-0003](../docs/adr/0003-rust-crate-frostyard-problems-in-rust-folder.md)
- Shared vectors: [problems/testdata/vectors/](../problems/testdata/vectors/README.md)

## Use

The crate is not on crates.io (`publish = false`). Add it as a cargo git
dependency on a std release tag:

```toml
[dependencies]
frostyard-problems = { git = "https://github.com/frostyard/std", tag = "vX.Y.Z" }
```

Make 1 recorder early in `main`, install the panic hook, and record coded
errors:

```rust
use frostyard_problems::{Component, Recorder};

fn main() {
    let rec = Recorder::new("org.frostyard.Frostmail", env!("CARGO_PKG_VERSION"), Component::Gui);
    if let Some(why) = rec.err() {
        eprintln!("problem records are off: {why}");
    }
    // Records each panic first, then calls the hook that was there before.
    rec.install_panic_hook(env!("CARGO_MANIFEST_DIR"));

    if let Err(e) = std::fs::read("/etc/frostmail.conf") {
        rec.error("config.load", Some(&e));
    }
}
```

- The app ID must be in the S8 allowlist and the version must be a release
  semver (`1.2.3` or `1.2.3-rc.1`); otherwise the recorder is inert.
- A code matches `^[a-z][a-z0-9_.]{0,63}$`; a bad code writes nothing.
- A panic location (`src/...rs:<line>`) is kept only for the crate that
  calls `install_panic_hook`: the function reads the file of its caller
  (for example `src-tauri/src/main.rs`) and matches it against
  `CARGO_MANIFEST_DIR` (for example `/b/app/src-tauri`) to find the crate
  root (`src-tauri/`); it keeps only locations under `<crate root>src/`.
  Call it directly in the app's crate with
  `env!("CARGO_MANIFEST_DIR")`. Other crates, dependencies and the standard
  library get no location, and an unknown or ambiguous root gives no
  location at all; see `map_location`.
- A `Display` impl that panics inside `error` gives the code as `MESSAGE`;
  `error` catches the panic and returns.
- The crate's own work in the panic hook never panics. A previous hook that
  panics or blocks is outside that guarantee.
- In the tests of an app, bind a `UnixDatagram` in a temporary folder, pass
  its path with `Recorder::with_socket`, and read a record back with
  `decode_entry` and `validate`.
- On an OS other than Linux the recorder writes nothing.

## Develop

Rust 1.99.1 with clippy, rustfmt and the Windows and macOS targets comes from
[`mise.toml`](../mise.toml) through `mise install`. From the repository root:

```sh
make rust-check   # fmt, clippy, tests, Windows and macOS check builds, no dependency
```
