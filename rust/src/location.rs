//! The location rule of a Rust panic: "our crates only".

use crate::patterns::{MAX_LOCATION_LEN, is_location};

/// Maps the file and line of a panic location to the value of
/// `FROSTYARD_LOCATION`, or `None` when the location must not leave the app.
///
/// The rule is "our crates only". It returns `Some("src/<rest>:<line>")` only
/// when all of these are true:
///
/// - `file` is a relative path: it does not start with `/`, and it holds no
///   `\` and no `:`.
/// - A path component of `file` is exactly `src`. The result starts at the
///   first such component: `src-tauri/src/sync/imap.rs` gives
///   `src/sync/imap.rs`.
/// - No component before that `src` looks like a cargo
///   `<name>-<major>.<minor>.<patch>` folder (for example `serde-1.0.219`),
///   the folder of a registry, vendored or `trim-paths` dependency.
/// - The result matches the v1 location pattern
///   `^src/([a-z0-9_]+/){0,8}[a-z0-9_]+\.rs:[1-9][0-9]{0,5}$` and is at most
///   128 bytes.
///
/// Why this works: cargo gives rustc the source path of a workspace member
/// relative to the workspace root, so `file!()` and a panic location of the
/// app's own crates are relative. Dependencies from crates.io, git and a
/// vendor folder (for example a Flatpak build) get an absolute path, and the
/// standard library gets `/rustc/<hash>/library/...`. All of those give
/// `None`, so no path of the build host and no file of another crate reach a
/// record.
///
/// Limits: the rule sees only the path. A path dependency inside the
/// workspace counts as one of our crates; a workspace folder named like
/// `<name>-1.2.3` before `src` hides the location; `--remap-path-prefix` or
/// `trim-paths` that make other paths relative to a folder with no version
/// in its name can let such a path through; and a location outside a `src`
/// folder (for example `build.rs`, `tests/` or `benches/`) gives `None`.
pub fn map_location(file: &str, line: u32) -> Option<String> {
    if file.starts_with('/') || file.contains('\\') || file.contains(':') {
        return None;
    }
    let mut offset = 0;
    for part in file.split('/') {
        if part == "src" {
            let out = format!("{}:{line}", file.get(offset..)?);
            return (out.len() <= MAX_LOCATION_LEN && is_location(&out)).then_some(out);
        }
        if is_crate_folder(part) {
            return None;
        }
        offset += part.len() + 1;
    }
    None
}

/// Reports whether `part` looks like a cargo `<name>-<major>.<minor>.<patch>`
/// folder: a `-` after a non-empty name, then 3 dot-separated numbers, then
/// the end or a `-` (prerelease) or `+` (build metadata).
fn is_crate_folder(part: &str) -> bool {
    part.match_indices('-')
        .any(|(i, _)| i > 0 && part.get(i + 1..).is_some_and(starts_with_version))
}

fn starts_with_version(v: &str) -> bool {
    let b = v.as_bytes();
    let mut i = 0;
    for k in 0..3 {
        let start = i;
        while b.get(i).is_some_and(u8::is_ascii_digit) {
            i += 1;
        }
        if i == start {
            return false;
        }
        if k < 2 {
            if b.get(i) != Some(&b'.') {
                return false;
            }
            i += 1;
        }
    }
    matches!(b.get(i), None | Some(b'-' | b'+'))
}
