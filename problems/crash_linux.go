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

// testAfterCreate, when not nil, runs after the own crash file is created
// and before it is locked. Only tests set it, to run a concurrent leftover
// scan in that window.
var testAfterCreate func(string)

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
// else $HOME/.local/state, then frostyard-problems/<appID>. The own file is
// created as <random>.crash-new, locked, given its header, and only then
// published as <random>.crash, so that a concurrent start never sees an
// unlocked *.crash file of a live process. Call it once,
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

	f, err := r.createCrashFile(dir)
	if err != nil {
		return err
	}
	if err := debug.SetCrashOutput(f, debug.CrashOptions{}); err != nil {
		return closeRemove(f, err)
	}
	crash.f = f
	return nil
}

// createCrashFile creates, locks and publishes the own crash file. The
// file starts as <random>.crash-new, a name that the leftover scan does not
// parse. Only after the lock is held and the header is written does it get
// its *.crash name, by a hard link that never replaces an existing file.
func (r *Recorder) createCrashFile(dir string) (*os.File, error) {
	for range crashCreateTries {
		f, err := os.CreateTemp(dir, "*"+crashNewSuffix) // mode 0600
		if err != nil {
			return nil, err
		}
		if testAfterCreate != nil {
			testAfterCreate(f.Name())
		}
		if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
			return nil, closeRemove(f, err)
		}
		if _, err := f.WriteString(crashHeader(r.appID, r.version, r.component)); err != nil {
			return nil, closeRemove(f, err)
		}
		tmp := f.Name()
		final := strings.TrimSuffix(tmp, crashNewSuffix) + crashSuffix
		err = os.Link(tmp, final)
		_ = os.Remove(tmp)
		if errors.Is(err, os.ErrExist) {
			// A crash file with that name exists: try a new name.
			_ = f.Close()
			continue
		}
		if err != nil {
			_ = f.Close()
			return nil, err
		}
		return f, nil
	}
	return nil, errors.New("problems: no free crash file name")
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
// skipped. It also removes, with no record, each *.crash-new file that is
// not locked and is older than crashNewStale: a process that died before it
// published its crash file. A younger one can belong to a live process that
// has not locked it yet.
func (r *Recorder) processLeftovers(dir string) {
	forEachCapped(dir, crashSuffix, r.processLeftover)
	forEachCapped(dir, crashNewSuffix, removeStaleNew)
}

// forEachCapped calls handle for each file in dir whose name ends with
// suffix, in name order, and stops after handle returned true
// crashMaxPerStart times.
func forEachCapped(dir, suffix string, handle func(string) bool) {
	paths, _ := filepath.Glob(filepath.Join(dir, "*"+suffix))
	slices.Sort(paths)
	done := 0
	for _, p := range paths {
		if done == crashMaxPerStart {
			return
		}
		if handle(p) {
			done++
		}
	}
}

// removeStaleNew removes 1 *.crash-new file of a dead process. It never
// reads the file.
func removeStaleNew(path string) bool {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return false
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil || now().Sub(fi.ModTime()) < crashNewStale {
		return false
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return false
	}
	_ = os.Remove(path)
	return true
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
