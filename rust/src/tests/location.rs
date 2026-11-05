//! `map_location`: the "our crates only" rule.

use crate::map_location;

#[test]
fn our_crates() {
    let cases = [
        ("src/main.rs", 12, "src/main.rs:12"),
        ("src/sync/imap.rs", 214, "src/sync/imap.rs:214"),
        ("src-tauri/src/sync/imap.rs", 7, "src/sync/imap.rs:7"),
        ("src-tauri/src/bridge.rs", 88, "src/bridge.rs:88"),
        ("crates/core/src/lib.rs", 1, "src/lib.rs:1"),
        ("src/a/src/b.rs", 3, "src/a/src/b.rs:3"),
        ("./src/main.rs", 5, "src/main.rs:5"),
        ("src/main.rs", 999_999, "src/main.rs:999999"),
    ];
    for (file, line, want) in cases {
        assert_eq!(
            map_location(file, line).as_deref(),
            Some(want),
            "map_location({file:?}, {line})"
        );
    }
}

#[test]
fn other_files_give_none() {
    let long = format!("src/{}.rs", "a".repeat(120));
    let cases: [(&str, u32); 22] = [
        // Absolute: crates.io, git and vendored dependencies, std.
        (
            "/home/u/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/serde-1.0.219/src/de/mod.rs",
            120,
        ),
        ("/home/u/.cargo/git/checkouts/foo-1a2b/abc123/src/lib.rs", 1),
        (
            "/rustc/0123456789abcdef0123456789abcdef01234567/library/core/src/option.rs",
            2000,
        ),
        ("/app/src/main.rs", 1),
        // trim-paths and vendored folders with a version.
        ("serde-1.0.219/src/de.rs", 10),
        ("vendor/serde-1.0.219/src/de.rs", 10),
        ("tokio-macros-2.5.0/src/lib.rs", 1),
        ("foo-1.0.0-rc.1/src/lib.rs", 1),
        ("foo-1.0.0+meta/src/lib.rs", 1),
        // Windows-like paths.
        ("src\\main.rs", 1),
        ("C:\\Users\\u\\app\\src\\main.rs", 1),
        ("C:/Users/u/app/src/main.rs", 1),
        // Lines out of the pattern.
        ("src/main.rs", 0),
        ("src/main.rs", 1_000_000),
        // Other shapes.
        ("src/Main.rs", 1),
        ("src/UI/main.rs", 1),
        ("src/main.go", 1),
        ("build.rs", 1),
        ("tests/it.rs", 1),
        ("source/main.rs", 1),
        ("src", 1),
        (long.as_str(), 1),
    ];
    for (file, line) in cases {
        assert_eq!(
            map_location(file, line),
            None,
            "map_location({file:?}, {line})"
        );
    }
}

#[test]
fn length_limit() {
    // "src/" + 119 + ".rs" + ":1" = 128 bytes: kept; 129 bytes (a longer
    // line or name): dropped.
    let at = |n: usize| format!("src/{}.rs", "a".repeat(n));
    assert!(map_location(&at(119), 1).is_some());
    assert_eq!(map_location(&at(119), 1).map(|s| s.len()), Some(128));
    assert_eq!(map_location(&at(119), 10), None);
    assert_eq!(map_location(&at(120), 1), None);
}

#[test]
fn version_like_names_after_src_do_not_count() {
    // Only folders before the first src component are checked.
    assert_eq!(
        map_location("src/v-1.2.3/main.rs", 1),
        None,
        "the pattern drops the '-' anyway"
    );
    assert_eq!(
        map_location("app-v2/src/main.rs", 4).as_deref(),
        Some("src/main.rs:4")
    );
    assert_eq!(
        map_location("app-1.2/src/main.rs", 4).as_deref(),
        Some("src/main.rs:4")
    );
}
