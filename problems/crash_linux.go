//go:build linux

package problems

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"syscall"
)

// crash is the crash output state of the process: the own crash file, kept
// open and locked for the life of the process.
var crash struct {
	mu sync.Mutex
	f  *os.File
}

// CrashOutput makes the Go runtime copy the report of a crash (an unrecovered
// panic in any goroutine, or a fatal error) to a crash file, through
// runtime/debug.SetCrashOutput. At the next start, CrashOutput turns each
// leftover crash file of a dead process into 1 record.
//
// The folder is $STATE_DIRECTORY (its first entry), else $XDG_STATE_HOME,
// else $HOME/.local/state, then frostyard-problems/<appID>. Call it once,
// early in main; a later call in the same process does nothing. On an inert
// Recorder it returns Err and does nothing. A crash is recorded only at the
// next start of the app.
func (r *Recorder) CrashOutput() error {
	if err := r.Err(); err != nil {
		return err
	}
	crash.mu.Lock()
	defer crash.mu.Unlock()
	if crash.f != nil {
		return nil
	}
	dir, err := crashDir(r.appID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	r.processLeftovers(dir)

	f, err := os.CreateTemp(dir, "*.crash") // mode 0600
	if err != nil {
		return err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return closeRemove(f, err)
	}
	if _, err := f.WriteString(crashHeader(r.appID, r.version, r.component)); err != nil {
		return closeRemove(f, err)
	}
	if err := debug.SetCrashOutput(f, debug.CrashOptions{}); err != nil {
		return closeRemove(f, err)
	}
	crash.f = f
	return nil
}

func closeRemove(f *os.File, err error) error {
	_ = f.Close()
	_ = os.Remove(f.Name())
	return err
}

// crashDir returns the crash folder of appID.
func crashDir(appID string) (string, error) {
	base := ""
	if s := os.Getenv("STATE_DIRECTORY"); s != "" {
		base, _, _ = strings.Cut(s, ":")
	}
	if !filepath.IsAbs(base) {
		base = os.Getenv("XDG_STATE_HOME")
	}
	if !filepath.IsAbs(base) {
		home := os.Getenv("HOME")
		if !filepath.IsAbs(home) {
			return "", errors.New("problems: no state folder: STATE_DIRECTORY, XDG_STATE_HOME and HOME are not set")
		}
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "frostyard-problems", appID), nil
}

// processLeftovers turns up to 8 crash files of dead processes into
// records, and removes them. A file that a live process holds locked is
// skipped.
func (r *Recorder) processLeftovers(dir string) {
	paths, _ := filepath.Glob(filepath.Join(dir, "*.crash"))
	slices.Sort(paths)
	done := 0
	for _, p := range paths {
		if done == crashMaxPerStart {
			return
		}
		if r.processLeftover(p) {
			done++
		}
	}
}

// processLeftover handles 1 crash file. It returns false when a live process
// owns the file, or when it cannot open it.
func (r *Recorder) processLeftover(path string) bool {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return false
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return false
	}
	data, err := io.ReadAll(io.LimitReader(f, crashReadMax))
	if err == nil {
		if c, ok := parseCrash(data); ok {
			hr := New(c.appID, c.version, c.component, WithSocket(r.socket))
			hr.recordFrames(c.message, c.frames)
		}
	}
	_ = os.Remove(path)
	return true
}

// markRecorded appends the recorded marker to the own crash file, when
// CrashOutput is active, so that the crash report of the re-panic that
// follows gives no second record.
func markRecorded() {
	crash.mu.Lock()
	defer crash.mu.Unlock()
	if crash.f != nil {
		_, _ = crash.f.WriteString(recordedMarker + "\n")
	}
}
