package problems

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// DefaultSocket is the journal socket that a Recorder writes to unless
// WithSocket names another one.
const DefaultSocket = "/run/systemd/journal/socket"

// Rate cap: at most rateMax records in any rateWindow, for each process.
const (
	rateMax    = 10
	rateWindow = 60 * time.Second
)

// status is the outcome of 1 attempt to write a record. Tests assert it.
type status int

const (
	statusSent            status = iota // the journal socket took the datagram
	statusDroppedTooBig                 // EMSGSIZE: the record does not fit in 1 datagram
	statusDroppedNoSocket               // ENOENT or ECONNREFUSED: no journal socket
	statusDroppedBusy                   // EAGAIN or ENOBUFS: the journal cannot take it now
	statusDroppedRate                   // the per-process rate cap
	statusDroppedOther                  // any other failure, or no transport on this OS
	statusInert                         // the recorder is inert (nil or a bad identity)
	statusInvalid                       // an argument fails its pattern
)

// now is the clock of the rate cap; tests replace it.
var now = time.Now

// rate is the per-process rate cap, shared by all recorders.
var rate struct {
	mu    sync.Mutex
	times []time.Time
}

// rateAllow reports whether 1 more record fits in the rate cap, and counts
// it when it does.
func rateAllow() bool {
	rate.mu.Lock()
	defer rate.mu.Unlock()
	t := now()
	keep := rate.times[:0]
	for _, x := range rate.times {
		if t.Sub(x) < rateWindow {
			keep = append(keep, x)
		}
	}
	rate.times = keep
	if len(rate.times) >= rateMax {
		return false
	}
	rate.times = append(rate.times, t)
	return true
}

// Option changes a Recorder made by New.
type Option func(*Recorder)

// WithSocket sets the path of the journal socket. The default is
// DefaultSocket. Apps use it in their tests.
func WithSocket(path string) Option {
	return func(r *Recorder) { r.socket = path }
}

// Recorder writes problem records of 1 app to the local journal. Make it
// with New. A nil *Recorder is valid and inert. A Recorder is safe for use
// by many goroutines.
type Recorder struct {
	appID     string
	version   string
	component Component
	socket    string
	err       error
}

var errNilRecorder = errors.New("problems: nil recorder")

// New returns a Recorder for the app appID at release version, in the part
// component. It never fails: when appID is not in the S8 allowlist, version
// is not a spec <semver>, or component is not in the S8 enum, the Recorder is
// inert (it writes nothing) and Err returns why.
func New(appID, version string, component Component, opts ...Option) *Recorder {
	r := &Recorder{appID: appID, version: version, component: component, socket: DefaultSocket}
	for _, o := range opts {
		o(r)
	}
	switch {
	case !appIDs[appID]:
		r.err = fmt.Errorf("problems: app id %q is not in the allowlist", appID)
	case !semverRe.MatchString(version):
		r.err = fmt.Errorf("problems: version %q is not a release semver", version)
	case !components[string(component)]:
		r.err = fmt.Errorf("problems: unknown component %q", component)
	}
	return r
}

// Err returns why the Recorder is inert, or nil when it writes records.
func (r *Recorder) Err() error {
	if r == nil {
		return errNilRecorder
	}
	return r.err
}

// Error records a coded error: class error, PRIORITY 3, FROSTYARD_CODE code,
// and MESSAGE err.Error() (the code when err is nil), truncated to 2048
// bytes. A code that fails ^[a-z][a-z0-9_.]{0,63}$ writes nothing. Error
// never blocks and never reports a failure to write.
func (r *Recorder) Error(code string, err error) {
	r.recordError(code, err)
}

func (r *Recorder) recordError(code string, err error) status {
	if r.Err() != nil {
		return statusInert
	}
	if !codeRe.MatchString(code) {
		return statusInvalid
	}
	msg := code
	if err != nil {
		// fmt catches a panic in a broken Error method.
		msg = fmt.Sprint(err)
	}
	return r.emit(r.fields("error", "3", msg, field{fieldCode, code}))
}

// RecoverPanic records a panic of the calling goroutine and then panics
// again with the same value. Use it only as `defer r.RecoverPanic()`: recover
// works only in a function that the defer statement calls directly. When
// there is no panic, it does nothing.
//
// The record has class panic, PRIORITY 2, MESSAGE the panic value as text,
// and FROSTYARD_FRAMES the stack, innermost first, at most 32 frames, each
// mapped by MapFrame. After the re-panic, Go prints the original panic with
// "[recovered, repanicked]" and the process exits with code 2.
//
// RecoverPanic covers only its own goroutine. It cannot catch a fatal error
// (concurrent map writes, out of memory, deadlock) or os.Exit; CrashOutput
// covers those at the next start.
func (r *Recorder) RecoverPanic() {
	v := recover()
	if v == nil {
		return
	}
	var pcs [2 * maxFrames]uintptr
	// Skip runtime.Callers and RecoverPanic.
	n := runtime.Callers(2, pcs[:])
	r.recordPanic(v, pcs[:n])
	panic(v)
}

func (r *Recorder) recordPanic(v any, pcs []uintptr) status {
	if r.Err() != nil {
		return statusInert
	}
	var frames []string
	iter := runtime.CallersFrames(pcs)
	for len(frames) < maxFrames {
		f, more := iter.Next()
		frames = append(frames, MapFrame(f.Function))
		if !more {
			break
		}
	}
	st := r.recordFrames(fmt.Sprint(v), frames)
	if st == statusSent {
		markRecorded()
	}
	return st
}

// recordFrames writes a panic record with already mapped frames.
func (r *Recorder) recordFrames(msg string, frames []string) status {
	if r.Err() != nil {
		return statusInert
	}
	if len(frames) == 0 || len(frames) > maxFrames {
		return statusInvalid
	}
	return r.emit(r.fields("panic", "2", msg, field{fieldFrames, strings.Join(frames, "\n")}))
}

// fields returns the journal fields of a record in the registry order.
func (r *Recorder) fields(class, priority, msg string, last field) []field {
	return []field{
		{fieldMessageID, MessageID},
		{fieldPriority, priority},
		{fieldMessage, truncate(msg, maxMessageLen)},
		{fieldSchema, Schema},
		{fieldAppID, r.appID},
		{fieldAppVersion, r.version},
		{fieldRuntime, "go"},
		{fieldComponent, string(r.component)},
		{fieldClass, class},
		last,
	}
}

// emit applies the rate cap and sends 1 datagram.
func (r *Recorder) emit(fields []field) status {
	if !rateAllow() {
		return statusDroppedRate
	}
	return sendDatagram(r.socket, encodeEntry(fields))
}

// truncate cuts s to at most n bytes, on a UTF-8 rune boundary. It backs up
// at most 3 bytes: a longer run of continuation bytes is not valid UTF-8, and
// then it cuts at n.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for i := n; i > n-utf8.UTFMax && i > 0; i-- {
		if utf8.RuneStart(s[i]) {
			return s[:i]
		}
	}
	return s[:n]
}
