//! The writer: identity, coded errors, panic records, the hook order, the
//! rate cap of the process, and the transport.

use std::cell::RefCell;
use std::fmt;
use std::path::Path;

use crate::recorder::{PanicView, Status, on_panic, payload_text};
use crate::tests::support::send_guard;
use crate::{Component, Recorder, decode_entry, validate};

fn nsl() -> Recorder {
    Recorder::new("org.frostyard.nsl", "1.2.3", Component::Cli)
}

/// A fake sender that keeps every datagram.
struct Capture(RefCell<Vec<Vec<u8>>>);

impl Capture {
    fn new() -> Capture {
        Capture(RefCell::new(Vec::new()))
    }

    fn send(&self) -> impl Fn(&Path, &[u8]) -> bool + '_ {
        |_, b| {
            self.0.borrow_mut().push(b.to_vec());
            true
        }
    }

    fn field(&self, i: usize, name: &str) -> Option<String> {
        let all = self.0.borrow();
        decode_entry(all.get(i)?).ok()?.get(name).cloned()
    }

    fn len(&self) -> usize {
        self.0.borrow().len()
    }
}

#[test]
fn recorder_is_send_sync_clone() {
    fn check<T: Send + Sync + Clone + 'static>() {}
    check::<Recorder>();
}

#[test]
fn component_names() {
    let all = [
        (Component::Daemon, "daemon"),
        (Component::Cli, "cli"),
        (Component::Gui, "gui"),
        (Component::Shell, "shell"),
        (Component::Other, "other"),
    ];
    for (c, name) in all {
        assert_eq!(c.as_str(), name);
    }
}

#[test]
fn inert_recorder_writes_nothing() {
    let _guard = send_guard();
    assert_eq!(nsl().err(), None);
    let bad_app = Recorder::new("org.example.app", "1.2.3", Component::Cli);
    let bad_version = Recorder::new("org.frostyard.nsl", "v1.2.3", Component::Cli);
    assert!(
        bad_app.err().is_some_and(|e| e.contains("allowlist")),
        "{:?}",
        bad_app.err()
    );
    assert!(
        bad_version.err().is_some_and(|e| e.contains("semver")),
        "{:?}",
        bad_version.err()
    );
    let cap = Capture::new();
    for rec in [bad_app, bad_version] {
        assert_eq!(rec.record_error("x", None, &cap.send()), Status::Inert);
        assert_eq!(
            rec.record_panic("boom", Some(""), Some(("src/main.rs", 1)), &cap.send()),
            Status::Inert
        );
    }
    assert_eq!(cap.len(), 0);
}

#[test]
fn bad_code_writes_nothing() {
    let _guard = send_guard();
    let cap = Capture::new();
    for code in ["", "Config.Load", "/home/brian/secret", "a-b"] {
        assert_eq!(
            nsl().record_error(code, None, &cap.send()),
            Status::Invalid,
            "{code:?}"
        );
    }
    assert_eq!(cap.len(), 0);
}

#[test]
fn error_message() {
    let _guard = send_guard();
    let cap = Capture::new();
    let err = std::io::Error::other("connection reset");
    assert_eq!(
        nsl().record_error("sync.imap", Some(&err), &cap.send()),
        Status::Sent
    );
    assert_eq!(
        nsl().record_error("sync.timeout", None, &cap.send()),
        Status::Sent
    );
    assert_eq!(cap.field(0, "MESSAGE").as_deref(), Some("connection reset"));
    assert_eq!(cap.field(0, "FROSTYARD_CODE").as_deref(), Some("sync.imap"));
    assert_eq!(cap.field(0, "PRIORITY").as_deref(), Some("3"));
    assert_eq!(cap.field(1, "MESSAGE").as_deref(), Some("sync.timeout"));
}

/// A Display impl that fails.
struct Broken;

impl fmt::Display for Broken {
    fn fmt(&self, _: &mut fmt::Formatter<'_>) -> fmt::Result {
        Err(fmt::Error)
    }
}

#[test]
fn failing_display_gives_the_code() {
    let _guard = send_guard();
    let cap = Capture::new();
    assert_eq!(
        nsl().record_error("x.y", Some(&Broken), &cap.send()),
        Status::Sent
    );
    assert_eq!(cap.field(0, "MESSAGE").as_deref(), Some("x.y"));
}

/// A Display impl that panics.
struct Panicking;

impl fmt::Display for Panicking {
    fn fmt(&self, _: &mut fmt::Formatter<'_>) -> fmt::Result {
        panic!("display boom")
    }
}

#[test]
fn panicking_display_gives_the_code() {
    let _guard = send_guard();
    let cap = Capture::new();
    assert_eq!(
        nsl().record_error("x.y", Some(&Panicking), &cap.send()),
        Status::Sent
    );
    assert_eq!(cap.len(), 1);
    assert_eq!(cap.field(0, "MESSAGE").as_deref(), Some("x.y"));
}

#[test]
fn message_is_cut_on_a_char_boundary() {
    let _guard = send_guard();
    let cap = Capture::new();
    let a = "a".repeat(2047);
    let msg = format!("{a}é and more");
    assert_eq!(
        nsl().record_error("big", Some(&msg), &cap.send()),
        Status::Sent
    );
    assert_eq!(cap.field(0, "MESSAGE"), Some(a.clone()));
    assert_eq!(
        nsl().record_panic(&msg, Some(""), None, &cap.send()),
        Status::Sent
    );
    assert_eq!(cap.field(1, "MESSAGE"), Some(a));
}

#[test]
fn panic_record_fields() {
    let _guard = send_guard();
    let cap = Capture::new();
    let rec = nsl();
    let tauri = Some("src-tauri/");
    let bridge = Some(("src-tauri/src/bridge.rs", 88));
    assert_eq!(
        rec.record_panic("boom", tauri, bridge, &cap.send()),
        Status::Sent
    );
    let std_file = Some(("/rustc/abc/library/core/src/option.rs", 2000));
    assert_eq!(
        rec.record_panic("boom", tauri, std_file, &cap.send()),
        Status::Sent
    );
    assert_eq!(
        rec.record_panic("boom", tauri, None, &cap.send()),
        Status::Sent
    );
    // No crate root: no location, also for a file of the app.
    assert_eq!(
        rec.record_panic("boom", None, bridge, &cap.send()),
        Status::Sent
    );
    assert_eq!(
        cap.field(0, "FROSTYARD_LOCATION").as_deref(),
        Some("src/bridge.rs:88")
    );
    assert_eq!(cap.field(0, "PRIORITY").as_deref(), Some("2"));
    assert_eq!(cap.field(0, "FROSTYARD_CLASS").as_deref(), Some("panic"));
    assert_eq!(cap.field(0, "FROSTYARD_RUNTIME").as_deref(), Some("rust"));
    for i in 1..4 {
        assert_eq!(cap.field(i, "FROSTYARD_LOCATION"), None, "record {i}");
    }
    for b in cap.0.borrow().iter() {
        let detail = validate(&decode_entry(b).expect("decode")).expect("validate");
        assert_eq!(detail.class, "panic");
    }
}

#[test]
fn rate_cap_of_the_process() {
    let _guard = send_guard();
    let cap = Capture::new();
    let (a, b) = (
        nsl(),
        Recorder::new("org.frostyard.Frostmail", "2.0.1", Component::Gui),
    );
    for i in 0..10 {
        let rec = if i % 2 == 0 { &a } else { &b };
        assert_eq!(
            rec.record_error("x", None, &cap.send()),
            Status::Sent,
            "record {i}"
        );
    }
    // The cap is shared by all recorders.
    assert_eq!(a.record_error("x", None, &cap.send()), Status::DroppedRate);
    assert_eq!(
        b.record_panic("boom", Some(""), None, &cap.send()),
        Status::DroppedRate
    );
    assert_eq!(cap.len(), 10);
}

#[test]
fn failed_send_is_dropped() {
    let _guard = send_guard();
    let refuse = |_: &Path, _: &[u8]| false;
    assert_eq!(nsl().record_error("x", None, &refuse), Status::Dropped);
}

/// A fake panic for the hook body.
struct FakePanic {
    message: &'static str,
    location: Option<(&'static str, u32)>,
}

impl PanicView for FakePanic {
    fn message(&self) -> &str {
        self.message
    }

    fn raw_location(&self) -> Option<(&str, u32)> {
        self.location
    }
}

#[test]
fn hook_records_before_the_previous_hook() {
    let _guard = send_guard();
    let events = RefCell::new(Vec::new());
    let send = |_: &Path, b: &[u8]| {
        let fields = decode_entry(b).expect("decode");
        events.borrow_mut().push(format!(
            "record {} {}",
            fields
                .get("MESSAGE")
                .map(String::as_str)
                .unwrap_or_default(),
            fields
                .get("FROSTYARD_LOCATION")
                .map(String::as_str)
                .unwrap_or("-"),
        ));
        true
    };
    let prev = || events.borrow_mut().push("previous".to_owned());
    let info = FakePanic {
        message: "boom",
        location: Some(("src/main.rs", 9)),
    };
    on_panic(&nsl(), Some(""), &info, &send, &prev);
    // Another crate root: the record has no location.
    on_panic(&nsl(), Some("src-tauri/"), &info, &send, &prev);
    let inert = Recorder::new("org.example.app", "1.2.3", Component::Cli);
    on_panic(&inert, Some(""), &info, &send, &prev);
    assert_eq!(
        events.into_inner(),
        vec![
            "record boom src/main.rs:9",
            "previous",
            "record boom -",
            "previous",
            "previous"
        ]
    );
}

#[test]
fn payload_as_text() {
    let s: Box<dyn std::any::Any + Send> = Box::new("static text");
    let owned: Box<dyn std::any::Any + Send> = Box::new(String::from("owned text"));
    let other: Box<dyn std::any::Any + Send> = Box::new(42_i32);
    assert_eq!(payload_text(s.as_ref()), "static text");
    assert_eq!(payload_text(owned.as_ref()), "owned text");
    assert_eq!(payload_text(other.as_ref()), "Box<dyn Any>");
}

#[cfg(target_os = "linux")]
mod transport {
    use super::nsl;
    use crate::recorder::Status;
    use crate::tests::support::{TestSocket, send_guard};
    use crate::transport::send_datagram;
    use crate::{DEFAULT_SOCKET, decode_entry};

    #[test]
    fn error_writes_1_datagram() {
        let _guard = send_guard();
        let sock = TestSocket::new();
        let err = std::io::Error::other("disk full");
        nsl()
            .with_socket(&sock.path)
            .error("store.write", Some(&err));
        let got = sock.recv_all();
        assert_eq!(got.len(), 1);
        let fields = decode_entry(&got[0]).expect("decode");
        assert_eq!(fields.get("MESSAGE").map(String::as_str), Some("disk full"));
    }

    /// `error()` returns normally after a panic in `Display`, and writes 1
    /// record with the code as `MESSAGE`.
    #[test]
    fn error_with_panicking_display_returns() {
        let _guard = send_guard();
        let sock = TestSocket::new();
        nsl()
            .with_socket(&sock.path)
            .error("store.write", Some(&super::Panicking));
        let got = sock.recv_all();
        assert_eq!(got.len(), 1);
        let fields = decode_entry(&got[0]).expect("decode");
        assert_eq!(
            fields.get("MESSAGE").map(String::as_str),
            Some("store.write")
        );
    }

    #[test]
    fn no_socket_is_no_error() {
        let _guard = send_guard();
        let sock = TestSocket::new();
        let rec = nsl().with_socket(sock.missing());
        rec.error("x", None);
        assert_eq!(rec.record_error("x", None, &send_datagram), Status::Dropped);
        assert!(sock.recv_all().is_empty());
    }

    #[test]
    fn too_big_is_dropped() {
        // A datagram larger than the socket buffer limit fails with EMSGSIZE.
        let sock = TestSocket::new();
        assert!(!send_datagram(&sock.path, &vec![b'a'; 16 << 20]));
        assert!(send_datagram(&sock.path, b"A=1\n"));
        assert_eq!(sock.recv_all().len(), 1);
    }

    #[test]
    fn default_socket_path() {
        assert_eq!(DEFAULT_SOCKET, "/run/systemd/journal/socket");
    }
}
