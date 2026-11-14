//! `validate`: the one validation path of a reader.

use std::collections::HashMap;
use std::fmt;

use crate::patterns::{
    MAX_FRAMES, MAX_LOCATION_LEN, is_allowed_frame, is_app_id, is_class, is_code, is_component,
    is_location, is_runtime, is_semver,
};
use crate::{MESSAGE_ID, SCHEMA};

/// Every `FROSTYARD_*` field that the journal record registry names. Any
/// other `FROSTYARD_*` field makes a record invalid.
const REGISTRY: [&str; 9] = [
    "FROSTYARD_SCHEMA",
    "FROSTYARD_APP_ID",
    "FROSTYARD_APP_VERSION",
    "FROSTYARD_RUNTIME",
    "FROSTYARD_COMPONENT",
    "FROSTYARD_CLASS",
    "FROSTYARD_CODE",
    "FROSTYARD_FRAMES",
    "FROSTYARD_LOCATION",
];

/// The S8 `app_problem` detail fields of a valid record.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct Detail {
    /// `app_id`: an app ID of the S8 allowlist.
    pub app_id: String,
    /// `app_version`: the release version, a spec `<semver>`.
    pub app_version: String,
    /// `runtime`: `go` or `rust`.
    pub runtime: String,
    /// `component`: `daemon`, `cli`, `gui`, `shell` or `other`.
    pub component: String,
    /// `class`: `panic` or `error`.
    pub class: String,
    /// `code`: the code of an error; `None` for a panic.
    pub code: Option<String>,
    /// `frames`: the frames of a Go panic, innermost first; empty for a Rust
    /// record, an error, or a Go panic with no frames.
    pub frames: Vec<String>,
    /// `location`: the location of a Rust panic, when it has one.
    pub location: Option<String>,
}

/// The error of [`validate`]: the record is not a valid record of schema
/// `frostyard.problem/1`, and a reader drops it.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Invalid(String);

impl Invalid {
    /// Why the record is invalid.
    pub fn reason(&self) -> &str {
        &self.0
    }
}

impl fmt::Display for Invalid {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "problems: invalid record: {}", self.0)
    }
}

impl std::error::Error for Invalid {}

fn invalid<T>(reason: impl Into<String>) -> Result<T, Invalid> {
    Err(Invalid(reason.into()))
}

/// Checks the journal fields of 1 record against schema v1 and the journal
/// record registry, and returns its S8 detail.
///
/// It has exactly the rules of the Go `Validate` (rule 7 of the spec). It
/// rejects: a `MESSAGE_ID` other than [`MESSAGE_ID`]; a `FROSTYARD_SCHEMA`
/// other than [`SCHEMA`]; a `FROSTYARD_*` field that the registry does not
/// name; an app ID off the allowlist; a version, code, frame or location that
/// fails its v1 pattern or length; a runtime, component or class outside its
/// enum (no mapping to `other`); `class=error` without a code, or with frames
/// or a location; `class=panic` with a code; `runtime=go` with a location;
/// `runtime=rust` with frames; an empty `FROSTYARD_FRAMES`, or more than 32
/// frames. It does not check `MESSAGE` or fields that do not start with
/// `FROSTYARD_`. Journal records are untrusted: a reader drops a record that
/// fails.
pub fn validate(fields: &HashMap<String, String>) -> Result<Detail, Invalid> {
    let get = |name: &str| fields.get(name).map(String::as_str);
    let text = |name: &str| get(name).unwrap_or_default().to_owned();

    if get("MESSAGE_ID") != Some(MESSAGE_ID) {
        return invalid("wrong MESSAGE_ID");
    }
    if let Some(name) = fields
        .keys()
        .find(|name| name.starts_with("FROSTYARD_") && !REGISTRY.contains(&name.as_str()))
    {
        return invalid(format!("field {name} is not in the registry"));
    }
    if get("FROSTYARD_SCHEMA") != Some(SCHEMA) {
        return invalid("unknown schema");
    }
    let mut d = Detail {
        app_id: text("FROSTYARD_APP_ID"),
        app_version: text("FROSTYARD_APP_VERSION"),
        runtime: text("FROSTYARD_RUNTIME"),
        component: text("FROSTYARD_COMPONENT"),
        class: text("FROSTYARD_CLASS"),
        ..Detail::default()
    };
    if !is_app_id(&d.app_id) {
        return invalid("app id not in the allowlist");
    }
    if !is_semver(&d.app_version) {
        return invalid("bad app version");
    }
    if !is_runtime(&d.runtime) {
        return invalid("unknown runtime");
    }
    if !is_component(&d.component) {
        return invalid("unknown component");
    }
    if !is_class(&d.class) {
        return invalid("unknown class");
    }

    let code = get("FROSTYARD_CODE");
    let frames = get("FROSTYARD_FRAMES");
    let location = get("FROSTYARD_LOCATION");

    match d.class.as_str() {
        "error" => {
            if code.is_none() {
                return invalid("error without a code");
            }
            if frames.is_some() || location.is_some() {
                return invalid("error with frames or location");
            }
        }
        _ => {
            if code.is_some() {
                return invalid("panic with a code");
            }
        }
    }
    if let Some(code) = code {
        if !is_code(code) {
            return invalid("bad code");
        }
        d.code = Some(code.to_owned());
    }
    if let Some(frames) = frames {
        if d.runtime != "go" {
            return invalid(format!("frames on a {} record", d.runtime));
        }
        if frames.is_empty() {
            return invalid("empty frames");
        }
        let list: Vec<&str> = frames.split('\n').collect();
        if list.len() > MAX_FRAMES {
            return invalid(format!("more than {MAX_FRAMES} frames"));
        }
        if list.iter().any(|f| !is_allowed_frame(f)) {
            return invalid("bad frame");
        }
        d.frames = list.into_iter().map(str::to_owned).collect();
    }
    if let Some(location) = location {
        if d.runtime != "rust" {
            return invalid(format!("location on a {} record", d.runtime));
        }
        if location.len() > MAX_LOCATION_LEN || !is_location(location) {
            return invalid("bad location");
        }
        d.location = Some(location.to_owned());
    }
    Ok(d)
}
