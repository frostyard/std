//! Source checks: the crate has no network code and no dependency.

use std::fs;
use std::path::{Path, PathBuf};

/// The names that network code needs. They are built with `concat!`, so
/// that this file does not hold them.
const FORBIDDEN: [&str; 4] = [
    concat!("std::", "net"),
    concat!("Tcp", "Stream"),
    concat!("Udp", "Socket"),
    concat!("ToSocket", "Addrs"),
];

fn rust_files(dir: &Path, out: &mut Vec<PathBuf>) {
    for entry in fs::read_dir(dir).expect("read the source folder") {
        let path = entry.expect("folder entry").path();
        if path.is_dir() {
            rust_files(&path, out);
        } else if path.extension().is_some_and(|e| e == "rs") {
            out.push(path);
        }
    }
}

#[test]
fn no_network_code() {
    let src = Path::new(env!("CARGO_MANIFEST_DIR")).join("src");
    let mut files = Vec::new();
    rust_files(&src, &mut files);
    assert!(files.len() >= 10, "found only {} source files", files.len());
    for file in files {
        let text = fs::read_to_string(&file).expect("read a source file");
        for name in FORBIDDEN {
            assert!(
                !text.contains(name),
                "{} holds {name}: the crate has no network code",
                file.display()
            );
        }
    }
}

#[test]
fn no_dependencies() {
    let manifest = Path::new(env!("CARGO_MANIFEST_DIR")).join("Cargo.toml");
    let text = fs::read_to_string(manifest).expect("read Cargo.toml");
    assert!(
        !text.contains(concat!("dependen", "cies")),
        "Cargo.toml names a dependency table: the crate uses only std"
    );
}
