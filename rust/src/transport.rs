//! The transport: 1 record is 1 datagram on a new non-blocking `AF_UNIX`
//! datagram socket. Only Linux has it.

use std::path::Path;

/// Sends `b` as 1 datagram to the socket at `path`, and reports whether the
/// socket took it.
///
/// The socket is new for each record and non-blocking: the call never waits,
/// never retries and never prints. Every error (no socket file, a full
/// socket buffer, a datagram that is too big) drops the record.
#[cfg(target_os = "linux")]
pub(crate) fn send_datagram(path: &Path, b: &[u8]) -> bool {
    use std::os::unix::net::UnixDatagram;

    let Ok(sock) = UnixDatagram::unbound() else {
        return false;
    };
    if sock.set_nonblocking(true).is_err() {
        return false;
    }
    sock.send_to(b, path).is_ok()
}

/// Sends nothing: this OS has no systemd journal.
#[cfg(not(target_os = "linux"))]
pub(crate) fn send_datagram(_path: &Path, _b: &[u8]) -> bool {
    false
}
