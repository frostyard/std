//! The per-process rate cap: at most 10 records in any 60 seconds, shared by
//! all recorders.

use std::collections::VecDeque;
use std::sync::{Mutex, PoisonError};
use std::time::{Duration, Instant};

/// At most this many records in any [`RATE_WINDOW`].
pub(crate) const RATE_MAX: usize = 10;
/// The window of the rate cap.
pub(crate) const RATE_WINDOW: Duration = Duration::from_secs(60);

/// A sliding window of the times of the last records.
#[derive(Debug, Default)]
pub(crate) struct RateCap {
    times: VecDeque<Instant>,
}

impl RateCap {
    pub(crate) const fn new() -> RateCap {
        RateCap {
            times: VecDeque::new(),
        }
    }

    /// Reports whether 1 more record at `now` fits in the cap, and counts it
    /// when it does. A record that does not fit is not counted.
    pub(crate) fn allow(&mut self, now: Instant) -> bool {
        self.times
            .retain(|t| now.saturating_duration_since(*t) < RATE_WINDOW);
        if self.times.len() >= RATE_MAX {
            return false;
        }
        self.times.push_back(now);
        true
    }
}

/// The cap of this process, shared by all recorders.
static RATE: Mutex<RateCap> = Mutex::new(RateCap::new());

/// Applies the process cap to 1 record now. A poisoned lock is used as is:
/// the panic hook must never panic.
pub(crate) fn rate_allow() -> bool {
    RATE.lock()
        .unwrap_or_else(PoisonError::into_inner)
        .allow(Instant::now())
}

/// Empties the process cap. Only tests call it.
#[cfg(test)]
pub(crate) fn rate_reset() {
    *RATE.lock().unwrap_or_else(PoisonError::into_inner) = RateCap::new();
}
