package problems

import (
	"net"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestSendNoSocket(t *testing.T) {
	resetRate(t)
	missing := filepath.Join(t.TempDir(), "missing")
	if st := sendDatagram(missing, []byte("A=1\n")); st != statusDroppedNoSocket {
		t.Errorf("ENOENT: status = %v, want dropped-no-socket", st)
	}
	// Error and RecoverPanic stay silent too.
	r := New("org.frostyard.nsl", "1.2.3", ComponentCLI, WithSocket(missing))
	if st := r.recordError("x", nil); st != statusDroppedNoSocket {
		t.Errorf("Error: status = %v, want dropped-no-socket", st)
	}

	// A socket file with no reader: ECONNREFUSED.
	path := filepath.Join(t.TempDir(), "closed")
	c, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: path, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	c.Close()
	if st := sendDatagram(path, []byte("A=1\n")); st != statusDroppedNoSocket {
		t.Errorf("closed socket: status = %v, want dropped-no-socket", st)
	}
}

func TestSendTooBig(t *testing.T) {
	sock, c := listen(t)
	testSndBuf = 1 // the kernel raises it to its minimum, about 4.5 KiB
	t.Cleanup(func() { testSndBuf = 0 })
	if st := sendDatagram(sock, make([]byte, 64<<10)); st != statusDroppedTooBig {
		t.Errorf("status = %v, want dropped-too-big", st)
	}
	expectNone(t, c)
}

func TestLargestRecordFits(t *testing.T) {
	sock, c := listen(t)
	resetRate(t)
	r := New("org.frostyard.Frostnotes", "999999.999999.999999-alpha.9999", ComponentOther, WithSocket(sock))
	frame := "github.com/frostyard/a." + strings.Repeat("x", maxFrameLen-len("github.com/frostyard/a."))
	frames := make([]string, maxFrames)
	for i := range frames {
		frames[i] = MapFrame(frame)
	}
	if frames[0] != frame {
		t.Fatalf("MapFrame(160 bytes) = %q", frames[0])
	}
	if st := r.recordFrames(strings.Repeat("\u20ac", 1000), frames); st != statusSent {
		t.Fatalf("status = %v, want sent", st)
	}
	rec := readRecord(t, c)
	if len(rec["MESSAGE"]) != 2046 {
		t.Errorf("MESSAGE has %d bytes, want 2046 (cut on a rune boundary)", len(rec["MESSAGE"]))
	}
	if _, err := Validate(rec); err != nil {
		t.Errorf("Validate = %v", err)
	}
}

func TestSendBusy(t *testing.T) {
	sock, _ := listen(t)
	testSndBuf = 1
	t.Cleanup(func() { testSndBuf = 0 })
	payload := make([]byte, 1000)
	for i := range 10000 {
		switch st := sendDatagram(sock, payload); st {
		case statusSent:
			continue
		case statusDroppedBusy:
			t.Logf("busy after %d datagrams", i)
			return
		default:
			t.Fatalf("status = %v, want sent or dropped-busy", st)
		}
	}
	t.Fatal("the socket never became busy")
}

func TestClassify(t *testing.T) {
	tests := map[syscall.Errno]status{
		syscall.EMSGSIZE:     statusDroppedTooBig,
		syscall.EAGAIN:       statusDroppedBusy,
		syscall.ENOBUFS:      statusDroppedBusy,
		syscall.ENOENT:       statusDroppedNoSocket,
		syscall.ECONNREFUSED: statusDroppedNoSocket,
		syscall.EACCES:       statusDroppedOther,
	}
	for errno, want := range tests {
		if got := classify(errno); got != want {
			t.Errorf("classify(%v) = %v, want %v", errno, got, want)
		}
	}
	if got := classify(nil); got != statusSent {
		t.Errorf("classify(nil) = %v, want sent", got)
	}
}
