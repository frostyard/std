package problems

import (
	"bytes"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// listen binds a unixgram test socket in a temporary folder.
func listen(t *testing.T) (string, *net.UnixConn) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "j")
	c, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: path, Net: "unixgram"})
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return path, c
}

// readDatagram returns the next datagram of the test socket.
func readDatagram(t *testing.T, c *net.UnixConn) []byte {
	t.Helper()
	if err := c.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 1<<20)
	n, err := c.Read(buf)
	if err != nil {
		t.Fatalf("read datagram: %v", err)
	}
	return buf[:n]
}

// readRecord returns the fields of the next datagram.
func readRecord(t *testing.T, c *net.UnixConn) map[string]string {
	t.Helper()
	m, err := DecodeEntry(readDatagram(t, c))
	if err != nil {
		t.Fatalf("DecodeEntry: %v", err)
	}
	return m
}

// expectNone fails when the test socket holds a datagram. A record is sent
// before the call that writes it returns, so a short wait is enough.
func expectNone(t *testing.T, c *net.UnixConn) {
	t.Helper()
	if err := c.SetReadDeadline(time.Now().Add(20 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 1<<20)
	n, err := c.Read(buf)
	if err == nil {
		t.Fatalf("got an unexpected datagram: %q", buf[:n])
	}
	var ne net.Error
	if !errors.As(err, &ne) || !ne.Timeout() {
		t.Fatalf("read: %v", err)
	}
}

// resetRate empties the rate cap and restores the clock after the test.
func resetRate(t *testing.T) {
	t.Helper()
	rate.mu.Lock()
	rate.times = nil
	rate.mu.Unlock()
	t.Cleanup(func() {
		now = time.Now
		rate.mu.Lock()
		rate.times = nil
		rate.mu.Unlock()
	})
}

func runWriteVector(t *testing.T, v vector) {
	t.Helper()
	resetRate(t)
	sock, c := listen(t)
	r := New(v.get("in.app_id"), v.get("in.version"), Component(v.get("in.component")), WithSocket(sock))
	var st status
	switch v.get("in.class") {
	case "error":
		var err error
		if msg, ok := v.kv["in.message"]; ok {
			err = errors.New(msg[0])
		}
		st = r.recordError(v.get("in.code"), err)
	case "panic":
		var frames []string
		for _, f := range v.kv["in.frame"] {
			frames = append(frames, MapFrame(f))
		}
		st = r.recordFrames(v.get("in.message"), frames)
	default:
		t.Fatalf("bad in.class %q", v.get("in.class"))
	}
	switch v.get("expect") {
	case "ok":
		if st != statusSent {
			t.Fatalf("status = %v, want sent", st)
		}
		if got, want := readDatagram(t, c), encodeEntry(v.journal); !bytes.Equal(got, want) {
			t.Errorf("datagram =\n%q\nwant\n%q", got, want)
		}
	case "reject":
		if st == statusSent {
			t.Errorf("status = sent, want nothing written")
		}
		expectNone(t, c)
	default:
		t.Fatalf("bad expect %q", v.get("expect"))
	}
}

// checkFrames fails when a frame holds a file path, a line, an offset or an
// argument, or fails MapFrame.
func checkFrames(t *testing.T, frames []string) {
	t.Helper()
	if len(frames) == 0 || len(frames) > maxFrames {
		t.Errorf("got %d frames", len(frames))
	}
	for _, f := range frames {
		if (strings.Contains(f, "/") && strings.Contains(f, ".go")) || strings.Contains(f, ":") ||
			strings.Contains(f, " +0x") || strings.Contains(f, "(0x") || MapFrame(f) != f {
			t.Errorf("bad frame %q", f)
		}
	}
}

func containsFrame(frames []string, name string) bool {
	for _, f := range frames {
		if f == name {
			return true
		}
	}
	return false
}
