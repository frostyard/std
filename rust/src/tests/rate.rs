//! The rate cap with given instants (no sleeps).

use std::time::{Duration, Instant};

use crate::rate::{RATE_MAX, RATE_WINDOW, RateCap};

#[test]
fn ten_in_a_window() {
    let t0 = Instant::now();
    let mut cap = RateCap::new();
    for _ in 0..RATE_MAX {
        assert!(cap.allow(t0));
    }
    assert!(!cap.allow(t0), "the 11th record in the window");
    assert!(!cap.allow(t0 + RATE_WINDOW - Duration::from_nanos(1)));
    // 60 seconds later the window is empty again.
    assert!(cap.allow(t0 + RATE_WINDOW));
}

#[test]
fn sliding_window() {
    let t0 = Instant::now();
    let s = Duration::from_secs;
    let mut cap = RateCap::new();
    for i in 0..10 {
        assert!(cap.allow(t0 + s(i)), "record {i}");
    }
    assert!(!cap.allow(t0 + s(30)));
    // A dropped record is not counted: at 60 s only the record of 0 s left.
    assert!(cap.allow(t0 + s(60)));
    assert!(!cap.allow(t0 + s(60)));
    assert!(cap.allow(t0 + s(61)));
    assert!(!cap.allow(t0 + s(61)));
}

#[test]
fn an_earlier_instant_counts_as_in_the_window() {
    let t0 = Instant::now() + Duration::from_secs(10);
    let mut cap = RateCap::new();
    for _ in 0..RATE_MAX {
        assert!(cap.allow(t0));
    }
    assert!(!cap.allow(t0 - Duration::from_secs(5)));
}
