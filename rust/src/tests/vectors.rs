//! The runner of the shared vectors in `problems/testdata/vectors/v1.txt`.

use std::collections::{BTreeMap, HashMap, HashSet};
use std::fmt::Display;
use std::fs;
use std::path::Path;

use crate::patterns::map_frame;
use crate::{Detail, validate};

/// 1 block of the vector file.
struct Vector {
    name: String,
    /// Every `key: value` line, in order.
    kv: Vec<(String, String)>,
    /// The `journal.*` lines in order; repeated `FROSTYARD_FRAMES` lines are
    /// joined with `\n`.
    journal: Vec<(String, String)>,
}

impl Vector {
    fn get(&self, key: &str) -> Option<&str> {
        self.kv
            .iter()
            .find(|(k, _)| k == key)
            .map(|(_, v)| v.as_str())
    }

    fn text(&self, key: &str) -> &str {
        self.get(key).unwrap_or_default()
    }

    fn all(&self, key: &str) -> Vec<String> {
        self.kv
            .iter()
            .filter(|(k, _)| k == key)
            .map(|(_, v)| v.clone())
            .collect()
    }
}

/// Parses the vector file in the format of its README.
fn parse(text: &str) -> Vec<Vector> {
    let mut out = Vec::new();
    let mut cur = Vector {
        name: String::new(),
        kv: Vec::new(),
        journal: Vec::new(),
    };
    let mut flush = |cur: &mut Vector| {
        if !cur.kv.is_empty() {
            assert!(!cur.name.is_empty(), "vector without a name: {:?}", cur.kv);
            out.push(std::mem::replace(
                cur,
                Vector {
                    name: String::new(),
                    kv: Vec::new(),
                    journal: Vec::new(),
                },
            ));
        }
    };
    for line in text.split('\n') {
        if line.is_empty() {
            flush(&mut cur);
            continue;
        }
        if line.starts_with('#') {
            continue;
        }
        let (key, value) = match line.split_once(": ") {
            Some(kv) => kv,
            None => match line.strip_suffix(':') {
                Some(key) => (key, ""),
                None => panic!("bad vector line {line:?}"),
            },
        };
        cur.kv.push((key.to_owned(), value.to_owned()));
        if key == "vector" {
            cur.name = value.to_owned();
        }
        if let Some(name) = key.strip_prefix("journal.") {
            match cur.journal.last_mut() {
                Some((last, joined)) if last == name && name == "FROSTYARD_FRAMES" => {
                    joined.push('\n');
                    joined.push_str(value);
                }
                _ => cur.journal.push((name.to_owned(), value.to_owned())),
            }
        }
    }
    flush(&mut cur);
    out
}

fn load() -> Vec<Vector> {
    let path = Path::new(env!("CARGO_MANIFEST_DIR")).join("../problems/testdata/vectors/v1.txt");
    let text = fs::read_to_string(&path).unwrap_or_else(|e| panic!("read {}: {e}", path.display()));
    parse(&text)
}

fn payload(v: &Vector) -> Detail {
    Detail {
        app_id: v.text("payload.app_id").to_owned(),
        app_version: v.text("payload.app_version").to_owned(),
        runtime: v.text("payload.runtime").to_owned(),
        component: v.text("payload.component").to_owned(),
        class: v.text("payload.class").to_owned(),
        code: v.get("payload.code").map(str::to_owned),
        frames: v.all("payload.frames"),
        location: v.get("payload.location").map(str::to_owned),
    }
}

fn check_validate(v: &Vector) -> Result<(), String> {
    let fields: HashMap<String, String> = v.journal.iter().cloned().collect();
    match (validate(&fields), v.text("expect")) {
        (Ok(d), "ok") => {
            let want = payload(v);
            if d == want {
                Ok(())
            } else {
                Err(format!("validate = {d:?}, want {want:?}"))
            }
        }
        (Err(e), "ok") => Err(format!("validate = {e}, want ok")),
        (Ok(d), "reject") => Err(format!(
            "validate accepted a record that the vector rejects: {d:?}"
        )),
        (Err(_), "reject") => Ok(()),
        (_, expect) => Err(format!("bad expect {expect:?}")),
    }
}

/// Gives the `in.*` values of a Rust `write` vector to the writer through a
/// test socket, and checks the datagram.
#[cfg(target_os = "linux")]
fn run_write(v: &Vector) -> Result<(), String> {
    use crate::Component;
    use crate::Recorder;
    use crate::journal::decode_ordered;
    use crate::tests::support::{TestSocket, send_guard};
    use crate::transport::send_datagram;

    let component = match v.text("in.component") {
        "daemon" => Component::Daemon,
        "cli" => Component::Cli,
        "gui" => Component::Gui,
        "shell" => Component::Shell,
        "other" => Component::Other,
        c => return Err(format!("a Rust writer has no component {c:?}")),
    };
    let _guard = send_guard();
    let sock = TestSocket::new();
    let rec =
        Recorder::new(v.text("in.app_id"), v.text("in.version"), component).with_socket(&sock.path);
    match v.text("in.class") {
        "error" => {
            let msg = v.get("in.message");
            let err: Option<&dyn Display> = msg.as_ref().map(|m| m as &dyn Display);
            rec.error(v.text("in.code"), err);
        }
        "panic" => {
            // in.location is the raw file:line of PanicHookInfo::location.
            let location = match v.get("in.location") {
                None => None,
                Some(raw) => {
                    let (file, line) = raw
                        .rsplit_once(':')
                        .ok_or_else(|| format!("in.location {raw:?} has no line"))?;
                    let line = line
                        .parse::<u32>()
                        .map_err(|e| format!("in.location {raw:?}: {e}"))?;
                    Some((file, line))
                }
            };
            // in.crate_root is the root that install_panic_hook derives from
            // the file of its caller; absent means "".
            let root = Some(v.text("in.crate_root"));
            let _ = rec.record_panic(v.text("in.message"), root, location, &send_datagram);
        }
        class => return Err(format!("bad in.class {class:?}")),
    }
    let got = sock.recv_all();
    match v.text("expect") {
        "ok" => {
            let [datagram] = got.as_slice() else {
                return Err(format!("got {} datagrams, want 1", got.len()));
            };
            let fields = decode_ordered(datagram).map_err(|e| e.to_string())?;
            if fields != v.journal {
                return Err(format!("written fields = {fields:?}, want {:?}", v.journal));
            }
            Ok(())
        }
        "reject" if got.is_empty() => Ok(()),
        "reject" => Err(format!("the writer wrote {} datagrams, want 0", got.len())),
        expect => Err(format!("bad expect {expect:?}")),
    }
}

#[test]
fn vectors() {
    let vectors = load();
    assert!(
        vectors.len() >= 40,
        "loaded {} vectors, want at least 40",
        vectors.len()
    );
    let mut counts: BTreeMap<&str, usize> = BTreeMap::new();
    let mut seen = HashSet::new();
    let mut rust_writes = 0;
    let mut failures = Vec::new();
    for v in &vectors {
        if !seen.insert(v.name.as_str()) {
            failures.push(format!("vector {}: name repeats", v.name));
        }
        let typ = v.text("type");
        *counts.entry(typ).or_default() += 1;
        let result = match typ {
            "frame" => {
                let (input, want) = (v.text("in.frame"), v.text("out.frame"));
                let got = map_frame(input);
                if got == want {
                    Ok(())
                } else {
                    Err(format!("map_frame({input:?}) = {got:?}, want {want:?}"))
                }
            }
            "write" => {
                let mut result = Ok(());
                if v.text("in.runtime") == "rust" {
                    rust_writes += 1;
                    #[cfg(target_os = "linux")]
                    {
                        result = run_write(v);
                    }
                }
                if result.is_ok() && v.text("expect") == "ok" {
                    result = check_validate(v);
                }
                result
            }
            "validate" => check_validate(v),
            other => Err(format!("unknown type {other:?}")),
        };
        if let Err(e) = result {
            failures.push(format!("vector {}: {e}", v.name));
        }
    }
    for typ in ["write", "validate", "frame"] {
        if counts.get(typ).copied().unwrap_or_default() == 0 {
            failures.push(format!("no {typ} vectors"));
        }
    }
    if rust_writes == 0 {
        failures.push("no write vectors with in.runtime: rust".to_owned());
    }
    assert!(failures.is_empty(), "{}", failures.join("\n"));
}

#[test]
fn parse_reads_the_format() {
    let text = "# header only\n\n# comment\nvector: a\ntype: validate\nempty:\nkey: x: y\n\
                journal.FROSTYARD_FRAMES: f1\njournal.FROSTYARD_FRAMES: f2\njournal.X: 1\n\
                payload.frames: f1\npayload.frames: f2\n\nvector: b\ntype: frame\n";
    let vs = parse(text);
    assert_eq!(vs.len(), 2);
    let a = &vs[0];
    assert_eq!(a.name, "a");
    assert_eq!(a.get("empty"), Some(""));
    assert_eq!(a.get("key"), Some("x: y"));
    assert_eq!(a.all("payload.frames"), vec!["f1", "f2"]);
    assert_eq!(
        a.journal,
        vec![
            ("FROSTYARD_FRAMES".to_owned(), "f1\nf2".to_owned()),
            ("X".to_owned(), "1".to_owned())
        ]
    );
    assert_eq!(vs[1].name, "b");
}

#[test]
#[should_panic(expected = "bad vector line")]
fn parse_rejects_a_line_without_a_key() {
    parse("vector: a\nno colon here\n");
}
