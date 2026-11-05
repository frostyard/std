package problems

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"
	"testing"
)

const testApp = "org.frostyard.nsl"

// resetCrash ends the crash output of the test process after the test.
func resetCrash(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		crash.mu.Lock()
		defer crash.mu.Unlock()
		if crash.f != nil {
			_ = debug.SetCrashOutput(nil, debug.CrashOptions{})
			_ = crash.f.Close()
			crash.f = nil
		}
	})
}

// clearStateEnv sets the state folder variables for 1 test.
func clearStateEnv(t *testing.T) {
	t.Helper()
	t.Setenv("STATE_DIRECTORY", "")
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "")
}

func crashFiles(t *testing.T, dir string) []string {
	t.Helper()
	m, err := filepath.Glob(filepath.Join(dir, "*.crash"))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func writeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCrashDir(t *testing.T) {
	clearStateEnv(t)
	if _, err := crashDir(testApp); err == nil {
		t.Error("crashDir with no variables: want an error")
	}
	t.Setenv("HOME", "relative")
	if _, err := crashDir(testApp); err == nil {
		t.Error("crashDir with a relative HOME: want an error")
	}
	t.Setenv("HOME", "/home/u")
	check := func(want string) {
		t.Helper()
		got, err := crashDir(testApp)
		if err != nil || got != want {
			t.Errorf("crashDir = %q, %v; want %q", got, err, want)
		}
	}
	check("/home/u/.local/state/frostyard-problems/" + testApp)
	t.Setenv("XDG_STATE_HOME", "/xdg")
	check("/xdg/frostyard-problems/" + testApp)
	t.Setenv("STATE_DIRECTORY", "relative")
	check("/xdg/frostyard-problems/" + testApp)
	t.Setenv("STATE_DIRECTORY", "/var/lib/a:/var/lib/b")
	check("/var/lib/a/frostyard-problems/" + testApp)
	t.Setenv("STATE_DIRECTORY", "/var/lib/a")
	check("/var/lib/a/frostyard-problems/" + testApp)
}

func TestProcessLeftovers(t *testing.T) {
	sock, c := listen(t)
	resetRate(t)
	dir := t.TempDir()
	r := New(testApp, "2.0.0", ComponentCLI, WithSocket(sock))

	writeFile(t, filepath.Join(dir, "a.crash"), crashFixturePanic)
	writeFile(t, filepath.Join(dir, "b.crash"), crashFixtureRepanic)
	writeFile(t, filepath.Join(dir, "c.crash"), strings.TrimPrefix(crashFixturePanic, crashFixtureHeader))
	writeFile(t, filepath.Join(dir, "d.crash"), crashFixtureHeader)
	writeFile(t, filepath.Join(dir, "e.crash"), strings.Replace(crashFixturePanic, testApp, "org.example.app", 1))
	writeFile(t, filepath.Join(dir, "keep.txt"), crashFixturePanic)

	// A file that a live process holds locked (another open file
	// description) is skipped.
	locked := filepath.Join(dir, "f.crash")
	writeFile(t, locked, crashFixturePanic)
	lf, err := os.Open(locked)
	if err != nil {
		t.Fatal(err)
	}
	defer lf.Close()
	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}

	// A symlink is not followed, and stays.
	if err := os.Symlink(filepath.Join(dir, "keep.txt"), filepath.Join(dir, "g.crash")); err != nil {
		t.Fatal(err)
	}
	// A folder that cannot be read is removed with no record.
	if err := os.Mkdir(filepath.Join(dir, "h.crash"), 0o700); err != nil {
		t.Fatal(err)
	}

	r.processLeftovers(dir)

	rec := readRecord(t, c)
	if rec["FROSTYARD_APP_VERSION"] != "1.2.3" || rec["FROSTYARD_COMPONENT"] != "daemon" || rec["MESSAGE"] != "panic: boom" {
		t.Errorf("record = %q, want the identity of the crash file header", rec)
	}
	if _, err := Validate(rec); err != nil {
		t.Errorf("Validate = %v", err)
	}
	checkFrames(t, strings.Split(rec["FROSTYARD_FRAMES"], "\n"))
	expectNone(t, c)

	got := crashFiles(t, dir)
	want := []string{locked, filepath.Join(dir, "g.crash")}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("crash files left = %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(dir, "keep.txt")); err != nil {
		t.Errorf("keep.txt: %v", err)
	}
}

func TestProcessLeftoversLimit(t *testing.T) {
	sock, c := listen(t)
	resetRate(t)
	dir := t.TempDir()
	for i := range 10 {
		writeFile(t, filepath.Join(dir, fmt.Sprintf("%02d.crash", i)), crashFixtureHeader)
	}
	New(testApp, "1.0.0", ComponentCLI, WithSocket(sock)).processLeftovers(dir)
	if got := len(crashFiles(t, dir)); got != 10-crashMaxPerStart {
		t.Errorf("%d files left, want %d", got, 10-crashMaxPerStart)
	}
	expectNone(t, c)
}

func TestCrashOutputInProcess(t *testing.T) {
	sock, c := listen(t)
	resetRate(t)
	resetCrash(t)
	clearStateEnv(t)
	state := t.TempDir()
	t.Setenv("STATE_DIRECTORY", state)
	dir := filepath.Join(state, "frostyard-problems", testApp)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "old.crash"), crashFixtureFatal)

	r := New(testApp, "1.4.0", ComponentGUI, WithSocket(sock))
	if err := r.CrashOutput(); err != nil {
		t.Fatalf("CrashOutput = %v", err)
	}
	if rec := readRecord(t, c); rec["MESSAGE"] != "fatal error: concurrent map writes" {
		t.Errorf("record = %q", rec)
	}
	expectNone(t, c)

	files := crashFiles(t, dir)
	if len(files) != 1 {
		t.Fatalf("crash files = %q, want 1 own file", files)
	}
	fi, err := os.Stat(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("crash file mode = %v, want 0600", fi.Mode().Perm())
	}
	data, _ := os.ReadFile(files[0])
	if string(data) != crashHeader(testApp, "1.4.0", ComponentGUI) {
		t.Errorf("crash file = %q", data)
	}

	// The own file is locked: a leftover scan skips it.
	r.processLeftovers(dir)
	if len(crashFiles(t, dir)) != 1 {
		t.Error("the own crash file was removed")
	}
	// A second call does nothing.
	if err := r.CrashOutput(); err != nil {
		t.Errorf("second CrashOutput = %v", err)
	}
	if len(crashFiles(t, dir)) != 1 {
		t.Error("a second call made a second crash file")
	}

	// RecoverPanic marks the own file after it records.
	func() {
		defer func() { _ = recover() }()
		defer r.RecoverPanic()
		panic("marked")
	}()
	readRecord(t, c)
	data, _ = os.ReadFile(files[0])
	if !strings.HasSuffix(string(data), recordedMarker+"\n") {
		t.Errorf("crash file = %q, want the recorded marker", data)
	}
}

func TestCrashOutputErrors(t *testing.T) {
	resetCrash(t)
	clearStateEnv(t)
	r := New(testApp, "1.0.0", ComponentCLI, WithSocket(filepath.Join(t.TempDir(), "none")))
	if err := r.CrashOutput(); err == nil {
		t.Error("CrashOutput with no state folder: want an error")
	}

	file := filepath.Join(t.TempDir(), "file")
	writeFile(t, file, "")
	t.Setenv("STATE_DIRECTORY", file)
	if err := r.CrashOutput(); err == nil {
		t.Error("CrashOutput with a file as state folder: want an error")
	}

	if os.Getuid() == 0 {
		t.Skip("root can write to a read-only folder")
	}
	state := t.TempDir()
	dir := filepath.Join(state, "frostyard-problems", testApp)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	t.Setenv("STATE_DIRECTORY", state)
	if err := r.CrashOutput(); err == nil {
		t.Error("CrashOutput with a read-only folder: want an error")
	}
	crash.mu.Lock()
	active := crash.f != nil
	crash.mu.Unlock()
	if active {
		t.Error("crash output is active after an error")
	}
}

func TestCloseRemove(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*.crash")
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Errorf("x")
	if err := closeRemove(f, want); err != want {
		t.Errorf("closeRemove = %v, want %v", err, want)
	}
	if _, err := os.Stat(f.Name()); !os.IsNotExist(err) {
		t.Errorf("file still there: %v", err)
	}
}

func TestCrashSubprocess(t *testing.T) {
	state := t.TempDir()
	dir := filepath.Join(state, "frostyard-problems", testApp)
	env := []string{"STATE_DIRECTORY=" + state, "PROBLEMS_TEST_SOCKET=" + filepath.Join(state, "none")}

	code, stderr := runHelper(t, "crash", env...)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stderr, "crash boom") || !strings.Contains(stderr, "crashingGoroutine") {
		t.Errorf("stderr:\n%s", stderr)
	}
	files := crashFiles(t, dir)
	if len(files) != 1 {
		t.Fatalf("crash files = %q, want 1", files)
	}
	data, _ := os.ReadFile(files[0])
	if !strings.HasPrefix(string(data), crashHeader(testApp, "1.2.3", ComponentCLI)+"panic: crash boom") {
		t.Errorf("crash file = %q", data)
	}

	// The next start turns the file into 1 record, with the version of the
	// run that crashed.
	sock, c := listen(t)
	code, stderr = runHelper(t, "leftover", "STATE_DIRECTORY="+state, "PROBLEMS_TEST_SOCKET="+sock, "PROBLEMS_TEST_VERSION=1.3.0")
	if code != 0 {
		t.Fatalf("leftover run: exit code = %d; stderr:\n%s", code, stderr)
	}
	rec := readRecord(t, c)
	if rec["FROSTYARD_APP_VERSION"] != "1.2.3" || rec["MESSAGE"] != "panic: crash boom" || rec["FROSTYARD_CLASS"] != "panic" {
		t.Errorf("record = %q", rec)
	}
	frames := strings.Split(rec["FROSTYARD_FRAMES"], "\n")
	checkFrames(t, frames)
	if !containsFrame(frames, "github.com/frostyard/std/problems.crashingGoroutine") {
		t.Errorf("frames %q do not hold crashingGoroutine", frames)
	}
	if _, err := Validate(rec); err != nil {
		t.Errorf("Validate = %v", err)
	}
	expectNone(t, c)
	if _, err := os.Stat(files[0]); !os.IsNotExist(err) {
		t.Errorf("the leftover file is still there: %v", err)
	}
}

func TestRecoverPanicCrashOutputSubprocess(t *testing.T) {
	state := t.TempDir()
	dir := filepath.Join(state, "frostyard-problems", testApp)
	sock, c := listen(t)
	env := []string{"STATE_DIRECTORY=" + state, "PROBLEMS_TEST_SOCKET=" + sock}

	code, stderr := runHelper(t, "recover-crash", env...)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr:\n%s", code, stderr)
	}
	readRecord(t, c)
	files := crashFiles(t, dir)
	if len(files) != 1 {
		t.Fatalf("crash files = %q, want 1", files)
	}
	data, _ := os.ReadFile(files[0])
	if !strings.Contains(string(data), recordedMarker+"\n") || !strings.Contains(string(data), "[recovered, repanicked]") {
		t.Errorf("crash file = %q, want the marker and the re-panic", data)
	}

	// The next start makes no second record.
	if code, stderr := runHelper(t, "leftover", env...); code != 0 {
		t.Fatalf("leftover run: exit code = %d; stderr:\n%s", code, stderr)
	}
	expectNone(t, c)
}
