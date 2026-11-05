//! The panic hook in a real panic: the test binary runs itself as a helper
//! process that installs the hook and panics.
#![cfg(target_os = "linux")]

use std::env;
use std::panic;
use std::process::{Command, Output};
use std::sync::atomic::Ordering;

use crate::journal::decode_ordered;
use crate::recorder::SENT;
use crate::tests::support::TestSocket;
use crate::{Component, MESSAGE_ID, Recorder, SCHEMA, decode_entry, validate};

/// Switches a run of the test binary into helper mode: `record` or `inert`.
const HELPER_ENV: &str = "FROSTYARD_PROBLEMS_TEST_HELPER";
/// The test socket path of the helper.
const SOCKET_ENV: &str = "FROSTYARD_PROBLEMS_TEST_SOCKET";
/// The previous hook of the helper prints this line, with the number of
/// records sent so far.
const MARKER: &str = "frostyard-problems-test: previous hook ran, sent=";

/// The line of the `panic!` in `boom`.
const BOOM_LINE: u32 = line!() + 3;
#[inline(never)]
fn boom() {
    panic!("helper boom");
}

/// Helper mode: a previous hook that prints the marker, then the recorder
/// hook, then a panic in this file (under `src/`).
fn helper(mode: &str) {
    let socket = env::var_os(SOCKET_ENV).expect("the helper needs a socket");
    panic::set_hook(Box::new(|_| {
        eprintln!("{MARKER}{}", SENT.load(Ordering::SeqCst));
    }));
    let app_id = if mode == "inert" {
        "org.example.app"
    } else {
        "org.frostyard.nsl"
    };
    Recorder::new(app_id, "1.2.3", Component::Cli)
        .with_socket(socket)
        .install_panic_hook();
    boom();
}

/// Runs the test `name` of this module again as a helper process.
fn run_helper(name: &str, mode: &str, sock: &TestSocket) -> Output {
    let module = module_path!()
        .split_once("::")
        .map(|(_, rest)| rest)
        .expect("module path");
    let exe = env::current_exe().expect("current exe");
    Command::new(exe)
        .args([
            "--exact",
            &format!("{module}::{name}"),
            "--nocapture",
            "--test-threads=1",
        ])
        .env(HELPER_ENV, mode)
        .env(SOCKET_ENV, &sock.path)
        .output()
        .expect("run the helper")
}

#[test]
fn panic_hook_records_then_calls_previous_hook() {
    if let Ok(mode) = env::var(HELPER_ENV) {
        helper(&mode);
        return;
    }
    let sock = TestSocket::new();
    let out = run_helper(
        "panic_hook_records_then_calls_previous_hook",
        "record",
        &sock,
    );
    let stdout = String::from_utf8_lossy(&out.stdout);
    let stderr = String::from_utf8_lossy(&out.stderr);
    assert!(
        !out.status.success() && stdout.contains("running 1 test"),
        "the helper must run 1 test and panic: {stdout}\n{stderr}"
    );
    // sent=1: the record was sent before the previous hook ran.
    assert!(
        stderr.contains(&format!("{MARKER}1")),
        "the previous hook did not run after the record: {stderr}"
    );
    let got = sock.recv_all();
    assert_eq!(got.len(), 1, "datagrams of the helper");
    let location = format!("{}:{BOOM_LINE}", file!());
    assert!(location.starts_with("src/"), "location {location}");
    let want: Vec<(String, String)> = [
        ("MESSAGE_ID", MESSAGE_ID),
        ("PRIORITY", "2"),
        ("MESSAGE", "helper boom"),
        ("FROSTYARD_SCHEMA", SCHEMA),
        ("FROSTYARD_APP_ID", "org.frostyard.nsl"),
        ("FROSTYARD_APP_VERSION", "1.2.3"),
        ("FROSTYARD_RUNTIME", "rust"),
        ("FROSTYARD_COMPONENT", "cli"),
        ("FROSTYARD_CLASS", "panic"),
        ("FROSTYARD_LOCATION", location.as_str()),
    ]
    .iter()
    .map(|(k, v)| ((*k).to_owned(), (*v).to_owned()))
    .collect();
    assert_eq!(decode_ordered(&got[0]).expect("decode"), want);
    let detail = validate(&decode_entry(&got[0]).expect("decode")).expect("validate");
    assert_eq!(detail.location.as_deref(), Some(location.as_str()));
}

#[test]
fn panic_hook_of_inert_recorder_calls_previous_hook() {
    if let Ok(mode) = env::var(HELPER_ENV) {
        helper(&mode);
        return;
    }
    let sock = TestSocket::new();
    let out = run_helper(
        "panic_hook_of_inert_recorder_calls_previous_hook",
        "inert",
        &sock,
    );
    let stdout = String::from_utf8_lossy(&out.stdout);
    let stderr = String::from_utf8_lossy(&out.stderr);
    assert!(
        !out.status.success() && stdout.contains("running 1 test"),
        "the helper must run 1 test and panic: {stdout}\n{stderr}"
    );
    assert!(
        stderr.contains(&format!("{MARKER}0")),
        "the previous hook did not run: {stderr}"
    );
    assert!(
        sock.recv_all().is_empty(),
        "an inert recorder wrote a record"
    );
}
