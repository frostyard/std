//! The native-protocol encoder and decoder, and the message cut.

use std::collections::HashMap;

use crate::decode_entry;
use crate::journal::{decode_ordered, encode_entry, truncate};

fn len_of(n: u64) -> Vec<u8> {
    n.to_le_bytes().to_vec()
}

fn cat(parts: &[&[u8]]) -> Vec<u8> {
    parts.concat()
}

#[test]
fn encode_decode_round_trip() {
    let fields = [
        ("MESSAGE", "line 1\nline 2=x"),
        ("FROSTYARD_FRAMES", "main.main\nruntime.main"),
        ("EMPTY", ""),
    ];
    let b = encode_entry(&fields);
    assert!(b.starts_with(b"MESSAGE\n"), "binary form");
    assert_eq!(b.last(), Some(&b'\n'));
    let want: Vec<(String, String)> = fields
        .iter()
        .map(|(k, v)| ((*k).to_owned(), (*v).to_owned()))
        .collect();
    assert_eq!(decode_ordered(&b).expect("decode"), want);
    let map: HashMap<String, String> = want.into_iter().collect();
    assert_eq!(decode_entry(&b).expect("decode"), map);
}

#[test]
fn decode_plain_and_mixed() {
    let b = cat(&[
        b"PRIORITY=3\nMESSAGE=a=b\n",
        &encode_entry(&[("CODE", "x\ny")]),
    ]);
    let got = decode_ordered(&b).expect("decode");
    assert_eq!(
        got,
        vec![
            ("PRIORITY".to_owned(), "3".to_owned()),
            ("MESSAGE".to_owned(), "a=b".to_owned()),
            ("CODE".to_owned(), "x\ny".to_owned()),
        ]
    );
}

#[test]
fn decode_rejects() {
    let cases: Vec<(&str, Vec<u8>)> = vec![
        ("empty", b"".to_vec()),
        ("no value", b"MESSAGE".to_vec()),
        ("lowercase name", b"message=x\n".to_vec()),
        ("empty name", b"=x\n".to_vec()),
        ("underscore name", b"_UID=0\n".to_vec()),
        ("repeated plain", b"A=1\nA=2\n".to_vec()),
        ("repeated binary", cat(&[b"A=1\nA\n", &len_of(1), b"2\n"])),
        ("plain no newline", b"A=1".to_vec()),
        ("short length", b"A\n\x01\x00".to_vec()),
        ("length too big", cat(&[b"A\n", &len_of(100), b"abc\n"])),
        (
            "length overflow",
            cat(&[b"A\n", &len_of(1 << 63), b"abc\n"]),
        ),
        ("length max", cat(&[b"A\n", &len_of(u64::MAX), b"abc\n"])),
        ("no final newline", cat(&[b"A\n", &len_of(3), b"abc"])),
        ("wrong final byte", cat(&[b"A\n", &len_of(2), b"abc\n"])),
        ("name with a space", b"A B=1\n".to_vec()),
        ("value not UTF-8", b"A=\xff\n".to_vec()),
        (
            "binary value not UTF-8",
            cat(&[b"A\n", &len_of(1), b"\xfe\n"]),
        ),
    ];
    for (name, input) in cases {
        let got = decode_entry(&input);
        assert!(
            got.is_err(),
            "{name}: decode_entry({input:?}) = {got:?}, want an error"
        );
    }
    let err = decode_entry(b"").expect_err("empty");
    assert_eq!(err.to_string(), "problems: empty entry");
}

#[test]
fn truncate_on_a_char_boundary() {
    let a = "a".repeat(2047);
    // A 2-byte char at bytes 2047-2048 goes.
    assert_eq!(truncate(&format!("{a}é"), 2048), a);
    // A 3-byte char at bytes 2046-2048 goes.
    let a46 = "a".repeat(2046);
    assert_eq!(truncate(&format!("{a46}€"), 2048), a46);
    // A 4-byte char at bytes 2045-2048 goes.
    let a45 = "a".repeat(2045);
    assert_eq!(truncate(&format!("{a45}😀"), 2048), a45);
    // A char that ends at the limit stays.
    let fits = format!("{a46}é");
    assert_eq!(fits.len(), 2048);
    assert_eq!(truncate(&format!("{fits}b"), 2048), fits);
    // Short text is not cut.
    assert_eq!(truncate("abc", 2048), "abc");
    assert_eq!(truncate(&"x".repeat(5000), 2048).len(), 2048);
}
