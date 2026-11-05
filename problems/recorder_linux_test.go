package problems

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewInert(t *testing.T) {
	sock, c := listen(t)
	resetRate(t)
	tests := map[string]*Recorder{
		"app id":    New("org.example.app", "1.2.3", ComponentCLI, WithSocket(sock)),
		"version":   New("org.frostyard.nsl", "v1.2.3", ComponentCLI, WithSocket(sock)),
		"component": New("org.frostyard.nsl", "1.2.3", Component("tray"), WithSocket(sock)),
		"nil":       nil,
	}
	for name, r := range tests {
		if r.Err() == nil {
			t.Errorf("%s: Err() = nil, want an error", name)
		}
		if st := r.recordError("x", nil); st != statusInert {
			t.Errorf("%s: status = %v, want inert", name, st)
		}
		r.Error("x", nil)
		if st := r.recordFrames("m", []string{"main.main"}); st != statusInert {
			t.Errorf("%s: recordFrames status = %v, want inert", name, st)
		}
		if st := r.recordPanic("v", nil); st != statusInert {
			t.Errorf("%s: recordPanic status = %v, want inert", name, st)
		}
		if err := r.CrashOutput(); err == nil {
			t.Errorf("%s: CrashOutput() = nil, want Err()", name)
		}
	}
	expectNone(t, c)

	if r := New("org.frostyard.nsl", "1.2.3", ComponentCLI); r.Err() != nil || r.socket != DefaultSocket {
		t.Errorf("New(valid) = %+v, want no error and the default socket", r)
	}
}

func TestErrorRecord(t *testing.T) {
	sock, c := listen(t)
	resetRate(t)
	r := New("org.frostyard.Frostread", "0.1.0-beta.3", ComponentGUI, WithSocket(sock))

	r.Error("config.load", errors.New("boom"))
	got := readRecord(t, c)
	want := map[string]string{
		"MESSAGE_ID": MessageID, "PRIORITY": "3", "MESSAGE": "boom", "FROSTYARD_SCHEMA": Schema,
		"FROSTYARD_APP_ID": "org.frostyard.Frostread", "FROSTYARD_APP_VERSION": "0.1.0-beta.3",
		"FROSTYARD_RUNTIME": "go", "FROSTYARD_COMPONENT": "gui", "FROSTYARD_CLASS": "error",
		"FROSTYARD_CODE": "config.load",
	}
	if len(got) != len(want) {
		t.Errorf("record has %d fields, want %d: %q", len(got), len(want), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	if _, err := Validate(got); err != nil {
		t.Errorf("Validate(own record) = %v", err)
	}

	// A nil error: MESSAGE is the code.
	r.Error("sync", nil)
	if got := readRecord(t, c); got["MESSAGE"] != "sync" {
		t.Errorf("MESSAGE = %q, want the code", got["MESSAGE"])
	}

	// A broken Error method does not panic.
	var be *brokenErr
	r.Error("broken", be)
	if got := readRecord(t, c); got["FROSTYARD_CODE"] != "broken" {
		t.Errorf("FROSTYARD_CODE = %q, want broken", got["FROSTYARD_CODE"])
	}

	// A bad code writes nothing.
	for _, code := range []string{"", "Upper", "1x", "a/b", "a" + strings.Repeat("b", 64)} {
		if st := r.recordError(code, nil); st != statusInvalid {
			t.Errorf("code %q: status = %v, want invalid", code, st)
		}
	}
	expectNone(t, c)
}

type brokenErr struct{ msg string }

func (e *brokenErr) Error() string { return e.msg }

func TestTruncate(t *testing.T) {
	tests := []struct {
		in   string
		n    int
		want string
	}{
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
		{"ab\u00e9", 3, "ab"},
		{"a\u20ac", 3, "a"},
		{"\u20ac\u20ac", 4, "\u20ac"},
		{"a\x80\x80\x80\x80\x80", 4, "a\x80\x80\x80"},
	}
	for _, tt := range tests {
		if got := truncate(tt.in, tt.n); got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.in, tt.n, got, tt.want)
		}
	}
	long := strings.Repeat("x", 3000)
	if got := truncate(long, maxMessageLen); len(got) != maxMessageLen {
		t.Errorf("len = %d, want %d", len(got), maxMessageLen)
	}
}

func TestRecordFramesBounds(t *testing.T) {
	r := New("org.frostyard.nsl", "1.2.3", ComponentCLI, WithSocket(filepath.Join(t.TempDir(), "none")))
	if st := r.recordFrames("m", nil); st != statusInvalid {
		t.Errorf("0 frames: status = %v, want invalid", st)
	}
	if st := r.recordFrames("m", make([]string, 33)); st != statusInvalid {
		t.Errorf("33 frames: status = %v, want invalid", st)
	}
}

func TestRecoverPanicInProcess(t *testing.T) {
	sock, c := listen(t)
	resetRate(t)
	r := New("org.frostyard.nsl", "1.2.3", ComponentDaemon, WithSocket(sock))

	var got any
	func() {
		defer func() { got = recover() }()
		defer r.RecoverPanic()
		panickingHelper()
	}()
	if got != "helper boom" {
		t.Fatalf("re-panic value = %v, want helper boom", got)
	}
	rec := readRecord(t, c)
	if rec["FROSTYARD_CLASS"] != "panic" || rec["PRIORITY"] != "2" || rec["MESSAGE"] != "helper boom" {
		t.Errorf("record = %q", rec)
	}
	if _, ok := rec["FROSTYARD_CODE"]; ok {
		t.Errorf("a panic record has FROSTYARD_CODE")
	}
	frames := strings.Split(rec["FROSTYARD_FRAMES"], "\n")
	checkFrames(t, frames)
	if !containsFrame(frames, "github.com/frostyard/std/problems.panickingHelper") {
		t.Errorf("frames %q do not hold panickingHelper", frames)
	}
	if containsFrame(frames, "github.com/frostyard/std/problems.(*Recorder).RecoverPanic") || containsFrame(frames, "runtime.Callers") {
		t.Errorf("frames %q hold the frames of RecoverPanic", frames)
	}
	if _, err := Validate(rec); err != nil {
		t.Errorf("Validate(own record) = %v", err)
	}

	// No panic: nothing.
	func() { defer r.RecoverPanic() }()
	expectNone(t, c)
}

func TestRecoverPanicNilRecorder(t *testing.T) {
	var r *Recorder
	var got any
	func() {
		defer func() { got = recover() }()
		defer r.RecoverPanic()
		panic("nil recorder")
	}()
	if got != "nil recorder" {
		t.Errorf("re-panic value = %v, want nil recorder", got)
	}
}

func TestRecoverPanicDeepStack(t *testing.T) {
	sock, c := listen(t)
	resetRate(t)
	r := New("org.frostyard.nsl", "1.2.3", ComponentDaemon, WithSocket(sock))
	func() {
		defer func() { _ = recover() }()
		defer r.RecoverPanic()
		deep(40)
	}()
	frames := strings.Split(readRecord(t, c)["FROSTYARD_FRAMES"], "\n")
	if len(frames) != maxFrames {
		t.Errorf("got %d frames, want %d", len(frames), maxFrames)
	}
}

//go:noinline
func deep(n int) {
	if n == 0 {
		panic("deep")
	}
	deep(n - 1)
}

func TestRateCap(t *testing.T) {
	sock, c := listen(t)
	resetRate(t)
	base := time.Unix(1_800_000_000, 0)
	cur := base
	now = func() time.Time { return cur }
	r := New("org.frostyard.nsl", "1.2.3", ComponentCLI, WithSocket(sock))
	for i := range rateMax {
		cur = base.Add(time.Duration(i) * time.Second)
		if st := r.recordError("x", nil); st != statusSent {
			t.Fatalf("record %d: status = %v, want sent", i+1, st)
		}
		readRecord(t, c)
	}
	// The cap is per process: another recorder shares it.
	other := New("org.frostyard.intuneme", "1.0.0", ComponentShell, WithSocket(sock))
	if st := other.recordError("x", nil); st != statusDroppedRate {
		t.Fatalf("record 11: status = %v, want dropped-rate", st)
	}
	expectNone(t, c)
	// 60 s after the first record, 1 slot is free again.
	cur = base.Add(rateWindow)
	if st := r.recordError("x", nil); st != statusSent {
		t.Errorf("after the window: status = %v, want sent", st)
	}
	readRecord(t, c)
	if st := r.recordError("x", nil); st != statusDroppedRate {
		t.Errorf("status = %v, want dropped-rate", st)
	}
	cur = base.Add(2 * rateWindow)
	if st := r.recordError("x", nil); st != statusSent {
		t.Errorf("after 2 windows: status = %v, want sent", st)
	}
	readRecord(t, c)
}

// runHelper re-runs the test binary as a helper program and returns its
// exit code and stderr.
func runHelper(t *testing.T, helper string, env ...string) (int, string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "PROBLEMS_TEST_HELPER="+helper)
	cmd.Env = append(cmd.Env, env...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	err := cmd.Run()
	var ee *exec.ExitError
	switch {
	case err == nil:
		return 0, stderr.String()
	case errors.As(err, &ee):
		return ee.ExitCode(), stderr.String()
	default:
		t.Fatalf("run helper: %v", err)
		return -1, ""
	}
}

func TestRecoverPanicSubprocess(t *testing.T) {
	sock, c := listen(t)
	code, stderr := runHelper(t, "recover", "PROBLEMS_TEST_SOCKET="+sock)
	if code != 2 {
		t.Errorf("exit code = %d, want 2; stderr:\n%s", code, stderr)
	}
	for _, s := range []string{"helper boom", "[recovered, repanicked]", "panickingHelper"} {
		if !strings.Contains(stderr, s) {
			t.Errorf("stderr does not hold %q:\n%s", s, stderr)
		}
	}
	rec := readRecord(t, c)
	frames := strings.Split(rec["FROSTYARD_FRAMES"], "\n")
	checkFrames(t, frames)
	if !containsFrame(frames, "github.com/frostyard/std/problems.panickingHelper") {
		t.Errorf("frames %q do not hold panickingHelper", frames)
	}
	if rec["MESSAGE"] != "helper boom" {
		t.Errorf("MESSAGE = %q", rec["MESSAGE"])
	}
	expectNone(t, c)
}
