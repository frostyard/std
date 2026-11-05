//! Problem records (panics and coded errors) of a frostyard app in the local
//! systemd journal.
//!
//! This crate is the Rust writer and validator of schema `frostyard.problem/1`
//! (kind S8 `app_problem` and the journal record registry of frostyard/core
//! `docs/specs/problem-report.md`). It follows the same contract as the Go
//! package `github.com/frostyard/std/problems`
//! (`docs/specs/problems-package.md` in frostyard/std, section "Rust crate")
//! and runs the same shared test vectors.
//!
//! The crate never sends anything over a network: a record is 1 datagram to
//! the local journal socket, and a record that does not fit, or that the
//! journal cannot take at once, is dropped. It uses only the Rust standard
//! library.
//!
//! ```no_run
//! use frostyard_problems::{Component, Recorder};
//!
//! let rec = Recorder::new("org.frostyard.Frostmail", "2.0.1", Component::Gui);
//! rec.install_panic_hook(env!("CARGO_MANIFEST_DIR"));
//! let err = std::io::Error::other("connection reset");
//! rec.error("sync.imap", Some(&err));
//! ```
//!
//! On an OS other than Linux, a [`Recorder`] writes nothing, and the panic
//! hook only calls the previous hook. [`validate`], [`map_location`] and
//! [`decode_entry`] work on every OS.
#![forbid(unsafe_code)]
#![warn(missing_docs)]

mod journal;
mod location;
mod patterns;
mod rate;
mod recorder;
mod transport;
mod validate;

#[cfg(test)]
mod tests;

pub use journal::{DecodeError, decode_entry};
pub use location::map_location;
pub use recorder::Recorder;
pub use validate::{Detail, Invalid, validate};

/// The journal `MESSAGE_ID` of every problem record.
pub const MESSAGE_ID: &str = "8915bdd215c84cbab0c31c05a1a33e7e";

/// The value of `FROSTYARD_SCHEMA`: the schema version whose patterns the
/// values of a record follow.
pub const SCHEMA: &str = "frostyard.problem/1";

/// The journal socket that a [`Recorder`] writes to, unless
/// [`Recorder::with_socket`] names another one.
pub const DEFAULT_SOCKET: &str = "/run/systemd/journal/socket";

/// The part of an app that records a problem (the S8 component enum).
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum Component {
    /// A background service (`daemon`).
    Daemon,
    /// A command-line tool (`cli`).
    Cli,
    /// A graphical app (`gui`).
    Gui,
    /// A shell extension or applet (`shell`).
    Shell,
    /// Any other part (`other`).
    Other,
}

impl Component {
    /// The value of `FROSTYARD_COMPONENT` for this component.
    pub const fn as_str(self) -> &'static str {
        match self {
            Component::Daemon => "daemon",
            Component::Cli => "cli",
            Component::Gui => "gui",
            Component::Shell => "shell",
            Component::Other => "other",
        }
    }
}
