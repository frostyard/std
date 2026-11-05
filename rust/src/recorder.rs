//! The writer: `Recorder`, coded errors and the panic hook.

use std::any::Any;
use std::fmt;
use std::panic::{self, Location, PanicHookInfo};
use std::path::{Path, PathBuf};

use crate::journal::{encode_entry, truncate};
use crate::location::{crate_root, map_location};
use crate::patterns::{MAX_MESSAGE_LEN, is_app_id, is_semver};
use crate::rate::rate_allow;
use crate::transport::send_datagram;
use crate::{Component, DEFAULT_SOCKET, MESSAGE_ID, SCHEMA};

/// The outcome of 1 attempt to write a record. Tests assert it.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum Status {
    /// The journal socket took the datagram.
    Sent,
    /// The send failed (no socket, a full buffer, too big, or no transport
    /// on this OS).
    Dropped,
    /// The per-process rate cap.
    DroppedRate,
    /// The recorder is inert.
    Inert,
    /// An argument fails its pattern (only `error` checks one, on Linux).
    #[cfg(any(test, target_os = "linux"))]
    Invalid,
}

/// A sender of 1 datagram to a socket path; it reports whether the socket
/// took it. The real one is the transport; tests give a fake.
pub(crate) type Sender<'a> = &'a dyn Fn(&Path, &[u8]) -> bool;

/// Counts the datagrams that a sender took, so that a test helper process
/// can tell whether the record came before the previous hook.
#[cfg(test)]
pub(crate) static SENT: std::sync::atomic::AtomicUsize = std::sync::atomic::AtomicUsize::new(0);

/// Writes problem records of 1 app to the local journal.
///
/// Make it with [`Recorder::new`]. A `Recorder` is `Send + Sync` and cheap to
/// clone; clones write the same records. Every write is 1 non-blocking
/// datagram: it never waits, never retries, and never reports a failure. At
/// most 10 records in any 60 seconds are written for each process, shared by
/// all recorders. On an OS other than Linux it writes nothing.
#[derive(Clone, Debug)]
pub struct Recorder {
    app_id: String,
    version: String,
    component: Component,
    socket: PathBuf,
    err: Option<String>,
}

impl Recorder {
    /// Returns a recorder for the app `app_id` at release `version`, in the
    /// part `component`.
    ///
    /// It never fails. When `app_id` is not in the S8 allowlist, or `version`
    /// does not match the v1 semver pattern (`1.2.3` or `1.2.3-rc.1`; no `v`
    /// prefix, no build metadata), the recorder is inert: it writes nothing,
    /// and [`Recorder::err`] says why.
    pub fn new(app_id: &str, version: &str, component: Component) -> Recorder {
        let err = if !is_app_id(app_id) {
            Some(format!(
                "problems: app id {app_id:?} is not in the allowlist"
            ))
        } else if !is_semver(version) {
            Some(format!(
                "problems: version {version:?} is not a release semver"
            ))
        } else {
            None
        };
        Recorder {
            app_id: app_id.to_owned(),
            version: version.to_owned(),
            component,
            socket: PathBuf::from(DEFAULT_SOCKET),
            err,
        }
    }

    /// Sets the path of the journal socket. The default is
    /// [`DEFAULT_SOCKET`]. Apps use it in their tests, with a datagram socket
    /// of their own and [`crate::decode_entry`].
    pub fn with_socket(mut self, path: impl Into<PathBuf>) -> Recorder {
        self.socket = path.into();
        self
    }

    /// Why the recorder is inert, or `None` when it writes records.
    pub fn err(&self) -> Option<&str> {
        self.err.as_deref()
    }

    /// Records a coded error: class `error`, `PRIORITY=3`,
    /// `FROSTYARD_CODE=code`, and `MESSAGE` the text of `err` (the code when
    /// `err` is `None`), cut to 2048 bytes.
    ///
    /// A code that fails `^[a-z][a-z0-9_.]{0,63}$` writes nothing. `MESSAGE`
    /// stays on the host; only the code can leave it. The call never blocks
    /// and never reports a failure to write.
    ///
    /// `err` is formatted only when a record can be written (a recorder that
    /// is not inert, a valid code, on Linux). When its `Display` impl returns
    /// an error or panics, `MESSAGE` is the code: the call catches the panic
    /// (`std::panic::catch_unwind`) and returns normally. A panic hook still
    /// sees that panic, so a hook from [`Recorder::install_panic_hook`] also
    /// writes a `panic` record for it. With `panic = "abort"` a panicking
    /// `Display` impl still aborts the process (the installed hook records
    /// that panic first). On an OS other than Linux the call returns at once
    /// and does not call `Display`.
    pub fn error(&self, code: &str, err: Option<&dyn fmt::Display>) {
        #[cfg(target_os = "linux")]
        let _ = self.record_error(code, err, &send_datagram);
        #[cfg(not(target_os = "linux"))]
        let _ = (code, err);
    }

    #[cfg(any(test, target_os = "linux"))]
    pub(crate) fn record_error(
        &self,
        code: &str,
        err: Option<&dyn fmt::Display>,
        send: Sender<'_>,
    ) -> Status {
        if self.err.is_some() {
            return Status::Inert;
        }
        if !crate::patterns::is_code(code) {
            return Status::Invalid;
        }
        let msg = match err {
            Some(err) => error_text(err).unwrap_or_else(|| code.to_owned()),
            None => code.to_owned(),
        };
        self.emit("error", "3", &msg, Some(("FROSTYARD_CODE", code)), send)
    }

    /// Records a panic: class `panic`, `PRIORITY=2`, `MESSAGE` the panic
    /// message, and `FROSTYARD_LOCATION` when there is a crate root and
    /// [`map_location`] maps the raw location with it.
    pub(crate) fn record_panic(
        &self,
        msg: &str,
        root: Option<&str>,
        location: Option<(&str, u32)>,
        send: Sender<'_>,
    ) -> Status {
        if self.err.is_some() {
            return Status::Inert;
        }
        let location = match (root, location) {
            (Some(root), Some((file, line))) => map_location(root, file, line),
            _ => None,
        };
        let last = location.as_deref().map(|l| ("FROSTYARD_LOCATION", l));
        self.emit("panic", "2", msg, last, send)
    }

    /// Applies the rate cap and sends 1 record with the fields in the order
    /// of the registry.
    fn emit(
        &self,
        class: &str,
        priority: &str,
        msg: &str,
        last: Option<(&str, &str)>,
        send: Sender<'_>,
    ) -> Status {
        let mut fields = vec![
            ("MESSAGE_ID", MESSAGE_ID),
            ("PRIORITY", priority),
            ("MESSAGE", truncate(msg, MAX_MESSAGE_LEN)),
            ("FROSTYARD_SCHEMA", SCHEMA),
            ("FROSTYARD_APP_ID", self.app_id.as_str()),
            ("FROSTYARD_APP_VERSION", self.version.as_str()),
            ("FROSTYARD_RUNTIME", "rust"),
            ("FROSTYARD_COMPONENT", self.component.as_str()),
            ("FROSTYARD_CLASS", class),
        ];
        fields.extend(last);
        if !rate_allow() {
            return Status::DroppedRate;
        }
        if send(&self.socket, &encode_entry(&fields)) {
            #[cfg(test)]
            SENT.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
            Status::Sent
        } else {
            Status::Dropped
        }
    }

    /// Installs a panic hook that records each panic, then calls the hook
    /// that was installed before.
    ///
    /// It takes the current hook with [`std::panic::take_hook`] and sets a
    /// new one with [`std::panic::set_hook`]. For each panic of any thread,
    /// the new hook FIRST writes 1 record (class `panic`, `PRIORITY=2`,
    /// `MESSAGE` the payload when it is a `&str` or a `String`, else
    /// `Box<dyn Any>`, and `FROSTYARD_LOCATION` when [`map_location`] maps
    /// the panic location), THEN calls the previous hook with the same
    /// [`PanicHookInfo`], so the usual panic output and any other hook still
    /// run. An inert recorder installs the hook too: it records nothing and
    /// still calls the previous hook.
    ///
    /// The location rule is "the crate that installs the hook". The function
    /// is `#[track_caller]`: it reads the file of its caller
    /// ([`std::panic::Location::caller`], for example `src/main.rs` or
    /// `src-tauri/src/main.rs`) and keeps the part before the first `src`
    /// component as the crate root (`""` or `src-tauri/`). A panic gets a
    /// location only in a file under `<crate root>src/`. When the caller file
    /// is absolute, holds a `\` or a `:`, or has no `src` component, no panic
    /// of the process gets a location. So call it directly in the app's own
    /// crate (in `main`), not through a helper in another crate.
    ///
    /// The work of this crate in the hook never panics: no unwrap, no
    /// indexing that can fail, and a poisoned lock is used as is. That
    /// guarantee does not cover the previous hook: a previous hook that
    /// panics aborts the process (a panic inside a panic hook aborts), and a
    /// previous hook that blocks blocks the panicking thread.
    ///
    /// Call it once, early in `main`. A hook that another library sets later
    /// replaces this one unless it chains to it the same way. With
    /// `panic = "abort"`, the hook still runs before the process aborts.
    /// Like [`std::panic::take_hook`], it panics when it is called from a
    /// panicking thread.
    #[track_caller]
    pub fn install_panic_hook(&self) {
        let root = crate_root(Location::caller().file());
        let prev = panic::take_hook();
        let rec = self.clone();
        panic::set_hook(Box::new(move |info| {
            on_panic(&rec, root.as_deref(), info, &send_datagram, &|| prev(info));
        }));
    }
}

/// The parts of a panic that the hook reads. [`PanicHookInfo`] gives them in
/// the real hook; tests give a fake.
pub(crate) trait PanicView {
    /// The panic message.
    fn message(&self) -> &str;
    /// The raw file and line of the panic.
    fn raw_location(&self) -> Option<(&str, u32)>;
}

impl PanicView for PanicHookInfo<'_> {
    fn message(&self) -> &str {
        payload_text(self.payload())
    }

    fn raw_location(&self) -> Option<(&str, u32)> {
        self.location().map(|l| (l.file(), l.line()))
    }
}

/// The text of a panic payload: a `&str` or a `String` as is, any other
/// payload `Box<dyn Any>` (as the default hook of std says).
pub(crate) fn payload_text(payload: &(dyn Any + Send)) -> &str {
    if let Some(s) = payload.downcast_ref::<&str>() {
        s
    } else if let Some(s) = payload.downcast_ref::<String>() {
        s.as_str()
    } else {
        "Box<dyn Any>"
    }
}

/// The body of the panic hook with the crate root `root`: record first,
/// then the previous hook. Tests run it with a given root.
pub(crate) fn on_panic(
    rec: &Recorder,
    root: Option<&str>,
    info: &dyn PanicView,
    send: Sender<'_>,
    prev: &dyn Fn(),
) {
    let _ = rec.record_panic(info.message(), root, info.raw_location(), send);
    prev();
}

/// The `Display` text of `err`, or `None` when its `Display` impl returns an
/// error or panics. A panic is caught here, so it never unwinds into the
/// app from [`Recorder::error`].
#[cfg(any(test, target_os = "linux"))]
pub(crate) fn error_text(err: &dyn fmt::Display) -> Option<String> {
    panic::catch_unwind(panic::AssertUnwindSafe(|| {
        let mut msg = String::new();
        fmt::write(&mut msg, format_args!("{err}"))
            .ok()
            .map(|()| msg)
    }))
    .ok()
    .flatten()
}
