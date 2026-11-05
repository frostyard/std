//go:build linux

package problems

import (
	"errors"
	"syscall"
)

// testSndBuf, when above 0, sets SO_SNDBUF of the sending socket. Only tests
// set it, to force EMSGSIZE or EAGAIN.
var testSndBuf int

// sendDatagram sends b as 1 datagram to the AF_UNIX/SOCK_DGRAM socket at
// path. The socket is non-blocking: it never waits, never retries and never
// falls back to a memfd. A record that the socket does not take is dropped.
func sendDatagram(path string, b []byte) status {
	fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM|syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		return statusDroppedOther
	}
	defer syscall.Close(fd)
	if testSndBuf > 0 {
		_ = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_SNDBUF, testSndBuf)
	}
	return classify(syscall.Sendto(fd, b, 0, &syscall.SockaddrUnix{Name: path}))
}

// classify maps the result of Sendto to a status.
func classify(err error) status {
	switch {
	case err == nil:
		return statusSent
	case errors.Is(err, syscall.EMSGSIZE):
		return statusDroppedTooBig
	case errors.Is(err, syscall.EAGAIN), errors.Is(err, syscall.ENOBUFS):
		return statusDroppedBusy
	case errors.Is(err, syscall.ENOENT), errors.Is(err, syscall.ECONNREFUSED):
		return statusDroppedNoSocket
	default:
		return statusDroppedOther
	}
}
