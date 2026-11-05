//! The v1 matchers against the examples of the Go tests and the edges of
//! each regular expression.

use crate::patterns::{
    is_app_id, is_class, is_code, is_component, is_frame, is_location, is_runtime, is_semver,
    map_frame,
};

#[test]
fn semver() {
    for ok in [
        "1.2.3",
        "0.0.0",
        "0.4.0-rc.2",
        "2.0.1",
        "999999.999999.999999",
        "1.2.3-alpha.0",
        "1.2.3-beta.9999",
        "10.20.30-rc.1",
    ] {
        assert!(is_semver(ok), "{ok:?} must match");
    }
    for bad in [
        "",
        "v1.2.3",
        "1.2",
        "1.2.3.4",
        "01.2.3",
        "1.02.3",
        "1.2.03",
        "1000000.0.0",
        "1.2.3+build.1",
        "1.2.3-dev.1",
        "1.2.3-rc",
        "1.2.3-rc.",
        "1.2.3-rc.01",
        "1.2.3-rc.10000",
        "1.2.3-rc.1.2",
        "1.2.3-",
        "1.2.3\n",
        " 1.2.3",
        "1..3",
        "-1.2.3",
        "1.2.3-RC.1",
        "１.2.3",
    ] {
        assert!(!is_semver(bad), "{bad:?} must not match");
    }
}

#[test]
fn code() {
    for ok in [
        "a",
        "a.b",
        "a.b_c",
        "io",
        "config.load",
        "sync.timeout",
        "x9",
        &format!("a{}", "b".repeat(63)),
    ] {
        assert!(is_code(ok), "{ok:?} must match");
    }
    for bad in [
        "",
        "Config.Load",
        "/home/brian/secret",
        "9a",
        "_a",
        ".a",
        "a-b",
        "a b",
        "a\n",
        "é",
        &format!("a{}", "b".repeat(64)),
    ] {
        assert!(!is_code(bad), "{bad:?} must not match");
    }
}

#[test]
fn location() {
    for ok in [
        "src/main.rs:1",
        "src/sync/imap.rs:214",
        "src/a/b/c/d/e/f/g/h/x.rs:999999",
        "src/a_1/b2.rs:10",
    ] {
        assert!(is_location(ok), "{ok:?} must match");
    }
    for bad in [
        "",
        "src/main.rs:0",
        "src/main.rs:01",
        "src/main.rs:1000000",
        "src/main.rs",
        "src/main.rs:",
        "src/main.rs:1:2",
        "/home/brian/src/main.rs:1",
        "main.rs:1",
        "src/Main.rs:1",
        "src/main.go:1",
        "src/main.rs.rs:1",
        "src/.rs:1",
        "src//main.rs:1",
        "src/a/b/c/d/e/f/g/h/i/x.rs:1",
        "src/a-b/main.rs:1",
        "src/main.rs:1\n",
        "lib/main.rs:1",
    ] {
        assert!(!is_location(bad), "{bad:?} must not match");
    }
}

#[test]
fn frame() {
    // The cases of the Go TestMapFrame.
    let cases = [
        (
            "github.com/frostyard/std/problems.(*Recorder).RecoverPanic",
            "github.com/frostyard/std/problems.(*Recorder).RecoverPanic",
        ),
        ("runtime.gopanic", "runtime.gopanic"),
        ("main.main.func1", "main.main.func1"),
        ("github.com/spf13/cobra.Execute", "other"),
        ("/home/u/main.go:12", "other"),
        ("main.f(0x1, 0x2)", "other"),
        ("panic", "other"),
        ("", "other"),
    ];
    for (input, want) in cases {
        assert_eq!(map_frame(input), want, "map_frame({input:?})");
    }
    let ok = format!("main.{}", "a".repeat(155));
    assert_eq!(map_frame(&ok), ok);
    assert_eq!(map_frame(&format!("main.{}", "a".repeat(156))), "other");

    for yes in [
        "other",
        "github.com/frostyard/a.F",
        "github.com/frostyard/A-b_c/d.F",
        "github.com/frostyard/a/b/c/d/e/f.F",
        "github.com/emersion/go-imap/v2.F",
        "github.com/emersion/go-imap/v2/imapclient.(*Client).Wait",
        "net/http.(*conn).serve",
        "go/ast.Walk",
        "internal/a/b/c/d.F",
        "main.F[...]",
        "main.a-b",
        "io.F.G",
    ] {
        assert!(is_frame(yes), "{yes:?} must match");
    }
    for no in [
        "github.com/frostyard.F",
        "github.com/frostyard/a/b/c/d/e/f/g.F",
        "github.com/frostyardx/a.F",
        "github.com/frostyard/a b.F",
        "internal/a/b/c/d/e.F",
        "net/HTTP.Serve",
        "iox.F",
        "io.",
        "io",
        "io/.F",
        "io//a.F",
        "io.F/G",
        "io.F G",
        "Other",
        "other.",
        "main.F\n",
        "gopkg.in/yaml.v3.(*decoder).unmarshal",
        "github.com/emersion/go-imap.F",
    ] {
        assert!(!is_frame(no), "{no:?} must not match");
    }
}

#[test]
fn closed_sets() {
    for id in [
        "org.frostyard.ChairLift",
        "org.frostyard.nsl",
        "org.frostyard.intuneme",
        "org.frostyard.pilothouse",
        "org.frostyard.Frostmail",
        "org.frostyard.Frostnotes",
        "org.frostyard.Frostread",
    ] {
        assert!(is_app_id(id), "{id}");
    }
    for id in [
        "",
        "org.example.app",
        "org.frostyard.chairlift",
        "org.frostyard.Evil",
    ] {
        assert!(!is_app_id(id), "{id}");
    }
    for c in ["daemon", "cli", "gui", "shell", "other"] {
        assert!(is_component(c), "{c}");
    }
    assert!(!is_component("tray"));
    assert!(is_runtime("go") && is_runtime("rust") && !is_runtime("python"));
    assert!(is_class("panic") && is_class("error") && !is_class("warning"));
}
