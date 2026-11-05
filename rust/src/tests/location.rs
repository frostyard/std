//! `map_location` and the crate root: the "crate that installs the hook"
//! rule.

use crate::location::crate_root;
use crate::map_location;

#[test]
fn root_from_the_caller_file() {
    let cases = [
        ("src/main.rs", Some("")),
        ("src-tauri/src/main.rs", Some("src-tauri/")),
        ("src/bin/x.rs", Some("")),
        ("crates/app/src/main.rs", Some("crates/app/")),
        ("src/a/src/b.rs", Some("")),
        ("/abs/src/main.rs", None),
        ("/rustc/abc/library/std/src/panicking.rs", None),
        ("examples/x.rs", None),
        ("build.rs", None),
        ("src", None),
        ("src\\main.rs", None),
        ("C:/app/src/main.rs", None),
        ("", None),
    ];
    for (file, want) in cases {
        assert_eq!(crate_root(file).as_deref(), want, "crate_root({file:?})");
    }
}

#[test]
fn files_of_the_crate() {
    let cases = [
        ("", "src/main.rs", 12, "src/main.rs:12"),
        ("", "src/sync/imap.rs", 214, "src/sync/imap.rs:214"),
        (
            "src-tauri/",
            "src-tauri/src/bridge.rs",
            88,
            "src/bridge.rs:88",
        ),
        ("crates/app/", "crates/app/src/lib.rs", 1, "src/lib.rs:1"),
        ("", "src/a/src/b.rs", 3, "src/a/src/b.rs:3"),
        ("", "src/main.rs", 999_999, "src/main.rs:999999"),
    ];
    for (root, file, line, want) in cases {
        assert_eq!(
            map_location(root, file, line).as_deref(),
            Some(want),
            "map_location({root:?}, {file:?}, {line})"
        );
    }
}

#[test]
fn other_files_give_none() {
    let long = format!("src/{}.rs", "a".repeat(120));
    let cases: [(&str, &str, u32); 28] = [
        // Absolute: crates.io, git and vendored dependencies, std.
        (
            "",
            "/home/u/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/serde-1.0.219/src/de/mod.rs",
            120,
        ),
        (
            "",
            "/home/u/.cargo/git/checkouts/foo-1a2b/abc123/src/lib.rs",
            1,
        ),
        (
            "",
            "/rustc/0123456789abcdef0123456789abcdef01234567/library/core/src/option.rs",
            2000,
        ),
        ("", "/app/src/main.rs", 1),
        // Relative paths of other crates.
        ("", "vendor/serde/src/de.rs", 10),
        ("", "library/core/src/option.rs", 10),
        ("", "serde-1.0.219/src/de.rs", 10),
        ("", "src-tauri/src/bridge.rs", 88),
        ("", "crates/core/src/lib.rs", 1),
        ("src-tauri/", "src/main.rs", 5),
        ("src-tauri/", "vendor/serde/src/de.rs", 10),
        ("src-tauri/", "src-tauri/vendor/x/src/a.rs", 1),
        ("src-tauri/", "other/src-tauri/src/a.rs", 1),
        ("", "./src/main.rs", 5),
        // A root that is not "" and does not end with '/'.
        ("src-tauri", "src-tauri/src/main.rs", 1),
        ("src-tauri", "src-taurisrc/main.rs", 1),
        // Windows-like paths.
        ("", "src\\main.rs", 1),
        ("", "C:\\Users\\u\\app\\src\\main.rs", 1),
        ("C:/app/", "C:/app/src/main.rs", 1),
        // Lines out of the pattern.
        ("", "src/main.rs", 0),
        ("", "src/main.rs", 1_000_000),
        // Other shapes.
        ("", "src/Main.rs", 1),
        ("", "src/UI/main.rs", 1),
        ("", "src/main.go", 1),
        ("", "build.rs", 1),
        ("", "tests/it.rs", 1),
        ("", "src", 1),
        ("", long.as_str(), 1),
    ];
    for (root, file, line) in cases {
        assert_eq!(
            map_location(root, file, line),
            None,
            "map_location({root:?}, {file:?}, {line})"
        );
    }
}

#[test]
fn length_limit() {
    // "src/" + 119 + ".rs" + ":1" = 128 bytes: kept; 129 bytes (a longer
    // line or name): dropped.
    let at = |n: usize| format!("src/{}.rs", "a".repeat(n));
    assert_eq!(map_location("", &at(119), 1).map(|s| s.len()), Some(128));
    assert_eq!(map_location("", &at(119), 10), None);
    assert_eq!(map_location("", &at(120), 1), None);
    // The root does not count: only the result is limited.
    let root = format!("{}/", "r".repeat(200));
    assert_eq!(
        map_location(&root, &format!("{root}{}", at(119)), 1).map(|s| s.len()),
        Some(128)
    );
}
