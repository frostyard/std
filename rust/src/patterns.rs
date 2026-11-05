//! The v1 patterns, the app ID allowlist and the enums of schema
//! `frostyard.problem/1`.
//!
//! The patterns are hand-written matchers (the crate has no regex
//! dependency). Each one gives exactly the result of the Go regular
//! expression of the same name in `problems/problems.go`, which is copied
//! from frostyard/core `docs/specs/schemas/problem-report/v1.json`. Go's `$`
//! without the `m` flag matches only at the end of the text, so no matcher
//! accepts a trailing newline.

/// At most this many frames in `FROSTYARD_FRAMES`.
pub(crate) const MAX_FRAMES: usize = 32;
/// At most this many bytes in 1 frame.
pub(crate) const MAX_FRAME_LEN: usize = 160;
/// At most this many bytes in `FROSTYARD_LOCATION`.
pub(crate) const MAX_LOCATION_LEN: usize = 128;
/// `MESSAGE` is cut to this many bytes.
pub(crate) const MAX_MESSAGE_LEN: usize = 2048;

/// The S8 app ID allowlist (closed).
const APP_IDS: [&str; 7] = [
    "org.frostyard.ChairLift",
    "org.frostyard.nsl",
    "org.frostyard.intuneme",
    "org.frostyard.pilothouse",
    "org.frostyard.Frostmail",
    "org.frostyard.Frostnotes",
    "org.frostyard.Frostread",
];

/// The roots of a Go frame other than `github.com/frostyard`, from the
/// alternation of the v1 frame pattern.
const FRAME_ROOTS: &[&str] = &[
    "github.com/emersion/go-imap/v2",
    "archive",
    "bufio",
    "bytes",
    "cmp",
    "compress",
    "container",
    "context",
    "crypto",
    "database",
    "debug",
    "embed",
    "encoding",
    "errors",
    "expvar",
    "flag",
    "fmt",
    "go",
    "hash",
    "html",
    "image",
    "index",
    "internal",
    "io",
    "iter",
    "log",
    "main",
    "maps",
    "math",
    "mime",
    "net",
    "os",
    "path",
    "plugin",
    "reflect",
    "regexp",
    "runtime",
    "slices",
    "sort",
    "strconv",
    "strings",
    "structs",
    "sync",
    "syscall",
    "testing",
    "text",
    "time",
    "unicode",
    "unique",
    "unsafe",
    "vendor",
    "weak",
];

pub(crate) fn is_app_id(s: &str) -> bool {
    APP_IDS.contains(&s)
}

pub(crate) fn is_component(s: &str) -> bool {
    matches!(s, "daemon" | "cli" | "gui" | "shell" | "other")
}

pub(crate) fn is_runtime(s: &str) -> bool {
    matches!(s, "go" | "rust")
}

pub(crate) fn is_class(s: &str) -> bool {
    matches!(s, "panic" | "error")
}

/// `0|[1-9][0-9]{0,max-1}`: a number of at most `max` digits with no leading
/// zero.
fn is_number(s: &str, max: usize) -> bool {
    match s.as_bytes() {
        [b'0'] => true,
        [b'1'..=b'9', rest @ ..] => rest.len() < max && rest.iter().all(u8::is_ascii_digit),
        _ => false,
    }
}

/// `[1-9][0-9]{0,max-1}`: a positive number of at most `max` digits.
fn is_positive(s: &str, max: usize) -> bool {
    s != "0" && is_number(s, max)
}

/// `[a-z0-9_]+`.
fn is_word(s: &str) -> bool {
    !s.is_empty()
        && s.bytes()
            .all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || c == b'_')
}

/// `[A-Za-z0-9_-]+`.
fn is_frostyard_segment(s: &str) -> bool {
    !s.is_empty()
        && s.bytes()
            .all(|c| c.is_ascii_alphanumeric() || c == b'_' || c == b'-')
}

/// `^(0|[1-9][0-9]{0,5})\.(0|[1-9][0-9]{0,5})\.(0|[1-9][0-9]{0,5})(-(alpha|beta|rc)\.(0|[1-9][0-9]{0,3}))?$`
pub(crate) fn is_semver(s: &str) -> bool {
    let (core, pre) = match s.split_once('-') {
        Some((core, pre)) => (core, Some(pre)),
        None => (s, None),
    };
    let mut parts = core.split('.');
    let core_ok = matches!(
        (parts.next(), parts.next(), parts.next(), parts.next()),
        (Some(a), Some(b), Some(c), None) if is_number(a, 6) && is_number(b, 6) && is_number(c, 6)
    );
    match pre {
        None => core_ok,
        Some(pre) => {
            core_ok
                && pre.split_once('.').is_some_and(|(tag, n)| {
                    matches!(tag, "alpha" | "beta" | "rc") && is_number(n, 4)
                })
        }
    }
}

/// `^[a-z][a-z0-9_.]{0,63}$`
pub(crate) fn is_code(s: &str) -> bool {
    let mut bytes = s.bytes();
    s.len() <= 64
        && bytes.next().is_some_and(|c| c.is_ascii_lowercase())
        && bytes.all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || c == b'_' || c == b'.')
}

/// `^src/([a-z0-9_]+/){0,8}[a-z0-9_]+\.rs:[1-9][0-9]{0,5}$`
pub(crate) fn is_location(s: &str) -> bool {
    let Some(rest) = s.strip_prefix("src/") else {
        return false;
    };
    // The path holds no ':', so the first ':' ends it.
    let Some((path, line)) = rest.split_once(':') else {
        return false;
    };
    if !is_positive(line, 6) {
        return false;
    }
    let (dirs, file) = match path.rsplit_once('/') {
        Some((dirs, file)) => (Some(dirs), file),
        None => (None, path),
    };
    if !file.strip_suffix(".rs").is_some_and(is_word) {
        return false;
    }
    match dirs {
        None => true,
        Some(dirs) => {
            let mut count = 0;
            for dir in dirs.split('/') {
                if !is_word(dir) {
                    return false;
                }
                count += 1;
            }
            count <= 8
        }
    }
}

/// The v1 frame pattern:
/// `^(other|(github\.com/frostyard(/[A-Za-z0-9_-]+){1,6}|(<roots>)(/[a-z0-9_]+){0,4})\.[A-Za-z0-9_.()*\[\]-]+)$`
///
/// No path segment holds a `.` or a `/`, and the name after the package
/// holds no `/`, so the first `.` after the root ends the package path: the
/// match needs no backtracking.
pub(crate) fn is_frame(s: &str) -> bool {
    if s == "other" {
        return true;
    }
    if s.strip_prefix("github.com/frostyard")
        .is_some_and(|rest| is_frame_tail(rest, 1, 6, is_frostyard_segment))
    {
        return true;
    }
    FRAME_ROOTS.iter().any(|root| {
        s.strip_prefix(root)
            .is_some_and(|rest| is_frame_tail(rest, 0, 4, is_word))
    })
}

/// `(/<segment>){min,max}\.[A-Za-z0-9_.()*\[\]-]+$`
fn is_frame_tail(rest: &str, min: usize, max: usize, segment: fn(&str) -> bool) -> bool {
    let Some((segments, name)) = rest.split_once('.') else {
        return false;
    };
    let name_ok = !name.is_empty()
        && name.bytes().all(|c| {
            c.is_ascii_alphanumeric()
                || matches!(c, b'_' | b'.' | b'(' | b')' | b'*' | b'[' | b']' | b'-')
        });
    if !name_ok {
        return false;
    }
    if segments.is_empty() {
        return min == 0;
    }
    let Some(segments) = segments.strip_prefix('/') else {
        return false;
    };
    let mut count = 0;
    for seg in segments.split('/') {
        if !segment(seg) {
            return false;
        }
        count += 1;
    }
    (min..=max).contains(&count)
}

/// The frame mapping of a writer (Go `MapFrame`): `name` when it matches the
/// v1 frame pattern and is at most 160 bytes, else `other`. The Rust writer
/// writes no frames; the tests run it against every `frame` vector, so that
/// the frame check of [`crate::validate`] is the same as in Go.
#[cfg(test)]
pub(crate) fn map_frame(name: &str) -> &str {
    if name.len() <= MAX_FRAME_LEN && is_frame(name) {
        name
    } else {
        "other"
    }
}
