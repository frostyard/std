//! Shared test helpers: the send lock and a test journal socket.

use std::sync::{Mutex, MutexGuard, PoisonError};

use crate::rate::rate_reset;

/// Serializes the tests that write records, because the rate cap is shared
/// by the whole process.
static SEND_LOCK: Mutex<()> = Mutex::new(());

/// Takes the send lock and empties the rate cap. Hold the guard for the whole
/// test.
pub(crate) fn send_guard() -> MutexGuard<'static, ()> {
    let guard = SEND_LOCK.lock().unwrap_or_else(PoisonError::into_inner);
    rate_reset();
    guard
}

#[cfg(target_os = "linux")]
pub(crate) use socket::TestSocket;

#[cfg(target_os = "linux")]
mod socket {
    use std::io::ErrorKind;
    use std::os::unix::net::UnixDatagram;
    use std::path::PathBuf;
    use std::sync::atomic::{AtomicUsize, Ordering};
    use std::time::{SystemTime, UNIX_EPOCH};
    use std::{env, fs, process};

    /// A datagram socket bound in a new folder under the temporary folder.
    /// Drop removes the folder.
    pub(crate) struct TestSocket {
        dir: PathBuf,
        /// The path of the socket.
        pub(crate) path: PathBuf,
        sock: UnixDatagram,
    }

    impl TestSocket {
        pub(crate) fn new() -> TestSocket {
            static COUNT: AtomicUsize = AtomicUsize::new(0);
            let nanos = SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .map(|d| d.subsec_nanos())
                .unwrap_or_default();
            let dir = env::temp_dir().join(format!(
                "frostyard-problems-test-{}-{}-{nanos}",
                process::id(),
                COUNT.fetch_add(1, Ordering::SeqCst)
            ));
            fs::create_dir(&dir).expect("make the test folder");
            let path = dir.join("journal.sock");
            let sock = UnixDatagram::bind(&path).expect("bind the test socket");
            sock.set_nonblocking(true)
                .expect("set the test socket non-blocking");
            TestSocket { dir, path, sock }
        }

        /// A path in the test folder with no socket file.
        pub(crate) fn missing(&self) -> PathBuf {
            self.dir.join("none.sock")
        }

        /// Every datagram that is in the socket buffer now.
        pub(crate) fn recv_all(&self) -> Vec<Vec<u8>> {
            let mut out = Vec::new();
            let mut buf = vec![0u8; 1 << 16];
            loop {
                match self.sock.recv(&mut buf) {
                    Ok(n) => out.push(buf[..n].to_vec()),
                    Err(e) if e.kind() == ErrorKind::WouldBlock => return out,
                    Err(e) => panic!("receive from the test socket: {e}"),
                }
            }
        }
    }

    impl Drop for TestSocket {
        fn drop(&mut self) {
            let _ = fs::remove_dir_all(&self.dir);
        }
    }
}
