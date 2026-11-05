//! The journal native protocol: the encoding of a record and the decoding of
//! 1 datagram.

use std::collections::{HashMap, HashSet};
use std::fmt;

/// Encodes `fields` in the binary form of the journal native protocol: for
/// each field, `NAME`, a newline, the value length as a little-endian u64,
/// the value, and a newline.
pub(crate) fn encode_entry(fields: &[(&str, &str)]) -> Vec<u8> {
    let size = fields
        .iter()
        .map(|(name, value)| name.len() + 1 + 8 + value.len() + 1)
        .sum();
    let mut b = Vec::with_capacity(size);
    for (name, value) in fields {
        b.extend_from_slice(name.as_bytes());
        b.push(b'\n');
        b.extend_from_slice(&(value.len() as u64).to_le_bytes());
        b.extend_from_slice(value.as_bytes());
        b.push(b'\n');
    }
    b
}

/// Cuts `s` to at most `n` bytes, on a char boundary. A `&str` is valid
/// UTF-8, so this backs up at most 3 bytes, as the Go writer does.
pub(crate) fn truncate(s: &str, n: usize) -> &str {
    if s.len() <= n {
        return s;
    }
    let mut i = n;
    while !s.is_char_boundary(i) {
        i -= 1;
    }
    s.get(..i).unwrap_or_default()
}

/// The error of [`decode_entry`]: the datagram is not 1 valid journal entry.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct DecodeError(String);

impl fmt::Display for DecodeError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "problems: {}", self.0)
    }
}

impl std::error::Error for DecodeError {}

fn fail<T>(msg: String) -> Result<T, DecodeError> {
    Err(DecodeError(msg))
}

/// Decodes 1 journal native protocol datagram into its fields, in order.
pub(crate) fn decode_ordered(mut b: &[u8]) -> Result<Vec<(String, String)>, DecodeError> {
    if b.is_empty() {
        return fail("empty entry".to_owned());
    }
    let mut out = Vec::new();
    let mut seen = HashSet::new();
    while !b.is_empty() {
        let Some(i) = b.iter().position(|&c| c == b'\n' || c == b'=') else {
            return fail("field without a value".to_owned());
        };
        let (name, sep, rest) = (&b[..i], b[i], &b[i + 1..]);
        let name_ok = name.first().is_some_and(|&c| c != b'_')
            && name
                .iter()
                .all(|&c| c.is_ascii_uppercase() || c.is_ascii_digit() || c == b'_');
        let name = String::from_utf8_lossy(name).into_owned();
        if !name_ok {
            return fail(format!("bad field name {name:?}"));
        }
        if !seen.insert(name.clone()) {
            return fail(format!("repeated field {name}"));
        }
        let value;
        if sep == b'=' {
            let Some(j) = rest.iter().position(|&c| c == b'\n') else {
                return fail(format!("field {name} has no final newline"));
            };
            value = &rest[..j];
            b = &rest[j + 1..];
        } else {
            let Some((len, rest)) = rest.split_first_chunk::<8>() else {
                return fail(format!("field {name} has a short length"));
            };
            let n = u64::from_le_bytes(*len);
            if n >= rest.len() as u64 || rest.get(n as usize) != Some(&b'\n') {
                return fail(format!("field {name} has a bad length"));
            }
            let n = n as usize;
            value = &rest[..n];
            b = &rest[n + 1..];
        }
        let Ok(value) = String::from_utf8(value.to_vec()) else {
            return fail(format!("field {name} is not UTF-8"));
        };
        out.push((name, value));
    }
    Ok(out)
}

/// Decodes 1 journal native protocol datagram, as the Go `DecodeEntry` does.
///
/// It reads the binary form (`NAME\n`, a little-endian u64 length, the value,
/// `\n`) and the plain form (`NAME=value\n`). It rejects an empty entry, a
/// repeated field, a bad length, a missing final newline, and a field name
/// that is not `^[A-Z0-9_]+$` or that starts with `_`. A value that is not
/// UTF-8 is also an error, because the fields are Rust `String`s. Tests of
/// apps use it to read a datagram of a test socket back.
pub fn decode_entry(b: &[u8]) -> Result<HashMap<String, String>, DecodeError> {
    decode_ordered(b).map(|fields| fields.into_iter().collect())
}
