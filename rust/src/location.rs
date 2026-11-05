//! The location rule of a Rust panic: "the crate that installs the hook".

use crate::patterns::{MAX_LOCATION_LEN, is_location};

/// Derives the crate root from the file of the code that installs the panic
/// hook: the relative prefix before its first `src` path component (`""` for
/// `src/main.rs`, `src-tauri/` for `src-tauri/src/main.rs`).
///
/// It returns `None` when `caller_file` is absolute, holds a `\` or a `:`,
/// or has no `src` folder. Then no panic of the process gets a location.
pub(crate) fn crate_root(caller_file: &str) -> Option<String> {
    if !is_relative(caller_file) {
        return None;
    }
    let mut offset = 0;
    for part in caller_file.split('/') {
        // A folder named src: the file name itself does not count.
        let is_folder = caller_file
            .get(offset + part.len()..)
            .is_some_and(|rest| rest.starts_with('/'));
        if part == "src" && is_folder {
            return caller_file.get(..offset).map(str::to_owned);
        }
        offset += part.len() + 1;
    }
    None
}

/// A relative path in the form that cargo gives a workspace member: no
/// leading `/`, no `\` and no `:` (no Windows path, no drive letter).
fn is_relative(file: &str) -> bool {
    !(file.starts_with('/') || file.contains('\\') || file.contains(':'))
}

/// Maps the file and line of a panic location to the value of
/// `FROSTYARD_LOCATION`, or `None` when the location must not leave the app.
///
/// The rule is "the crate that installs the hook". `crate_root` is the
/// relative folder of that crate in the cargo workspace: `""` when the crate
/// is the workspace root, else a prefix that ends with `/` (for example
/// `src-tauri/`). [`crate::Recorder::install_panic_hook`] derives it from the
/// file of its caller. The function returns `Some("src/<rest>:<line>")` only
/// when all of these are true:
///
/// - `crate_root` is `""` or ends with `/`.
/// - `file` is a relative path: it does not start with `/`, and it holds no
///   `\` and no `:`.
/// - `file` starts exactly with `crate_root` followed by `src/`: the file is
///   in the `src` folder of that crate.
/// - The result `src/<rest>:<line>` matches the v1 location pattern
///   `^src/([a-z0-9_]+/){0,8}[a-z0-9_]+\.rs:[1-9][0-9]{0,5}$` and is at most
///   128 bytes.
///
/// Why this works: cargo gives rustc the source path of a workspace member
/// relative to the workspace root, so the files of the app's crate all start
/// with the same prefix as the file that installs the hook. Dependencies from
/// crates.io, git and a vendor folder get an absolute path, and the standard
/// library gets `/rustc/<hash>/library/...`. A relative path of another
/// crate (another workspace member, `vendor/<crate>/src/...` inside the
/// workspace, or `library/core/src/...` after a path remapping) does not
/// start with `<crate_root>src/`. So only the files of the app's own crate
/// reach a record.
///
/// Limits: other crates of the same workspace get no location; a panic in a
/// file of the app's crate outside `src/` (`build.rs`, `tests/`, `benches/`)
/// gets none; and when a path remapping (`--remap-path-prefix`,
/// `trim-paths`) makes the app's paths absolute or changes their prefix, no
/// panic gets a location.
pub fn map_location(crate_root: &str, file: &str, line: u32) -> Option<String> {
    if !(crate_root.is_empty() || crate_root.ends_with('/')) || !is_relative(file) {
        return None;
    }
    let rest = file.strip_prefix(crate_root)?.strip_prefix("src/")?;
    let out = format!("src/{rest}:{line}");
    (out.len() <= MAX_LOCATION_LEN && is_location(&out)).then_some(out)
}
