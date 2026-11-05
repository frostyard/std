package problems

import (
	"slices"
	"strings"
	"time"
)

// Crash file format: the header line, then whatever the Go runtime writes
// through runtime/debug.SetCrashOutput (a traceback), and maybe the marker.
const (
	crashHeaderVersion = "frostyard-problems/1"
	recordedMarker     = "frostyard-problems: recorded"
	crashReadMax       = 64 << 10
	crashMaxPerStart   = 8
	crashSuffix        = ".crash"
	// crashNewSuffix names an own crash file before it is locked and
	// published; the leftover scan never parses it.
	crashNewSuffix = ".crash-new"
	// crashNewStale is the age after which an unlocked *.crash-new file
	// belongs to a dead process.
	crashNewStale    = time.Hour
	crashCreateTries = 8
	// repanicSuffix ends the panic line that the Go runtime prints when a
	// recovered panic panics again with the same value.
	repanicSuffix = " [recovered, repanicked]"
)

// crashHeader returns the header line of a crash file.
func crashHeader(appID, version string, component Component) string {
	return crashHeaderVersion + " app=" + appID + " version=" + version + " component=" + string(component) + "\n"
}

// crashReport is the record that a leftover crash file turns into.
type crashReport struct {
	appID, version string
	component      Component
	message        string
	frames         []string
}

// parseCrash reads a leftover crash file. It returns false when the file
// gives no record: no valid header, no Go traceback (a panic or fatal error
// line and a goroutine block with at least 1 frame), or a crash that
// RecoverPanic already recorded. That is a file with the recorded marker
// whose first panic message ends with " [recovered, repanicked]"; a fatal
// error or a plain panic gives a record even when the marker is there,
// because other code (for example net/http) can recover the re-panic and the
// process can crash later.
//
// The Go runtime prints a multi-line panic value as the "panic: " line and 1
// tab-indented continuation line for each further line; the suffix goes on
// the last line. It also prints each nested panic as a tab-indented
// "panic: " line (printpanics). The message is all these lines, joined by
// "\n", with the tabs removed. A block with a nested "panic: " line is never
// skipped: RecoverPanic re-panics only 1 value, and an old marker can be in
// the file of a later crash. A duplicate record is better than a lost one.
// Only the first goroutine block gives frames; file paths, line numbers,
// offsets and arguments never reach a frame.
func parseCrash(data []byte) (crashReport, bool) {
	var c crashReport
	lines := strings.Split(string(data), "\n")
	h := strings.Fields(lines[0])
	if len(h) != 4 || h[0] != crashHeaderVersion {
		return c, false
	}
	app, ok1 := strings.CutPrefix(h[1], "app=")
	version, ok2 := strings.CutPrefix(h[2], "version=")
	component, ok3 := strings.CutPrefix(h[3], "component=")
	if !ok1 || !ok2 || !ok3 {
		return c, false
	}
	c.appID, c.version, c.component = app, version, Component(component)

	i := 1
	for ; i < len(lines); i++ {
		l := lines[i]
		if strings.HasPrefix(l, "fatal error: ") {
			c.message = truncate(l, maxMessageLen)
			break
		}
		if strings.HasPrefix(l, "panic: ") {
			msg := []string{l}
			nested := false
			for i+1 < len(lines) && strings.HasPrefix(lines[i+1], "\t") {
				i++
				msg = append(msg, lines[i][1:])
				nested = nested || strings.HasPrefix(lines[i][1:], "panic: ")
			}
			if !nested && strings.HasSuffix(msg[len(msg)-1], repanicSuffix) && slices.Contains(lines, recordedMarker) {
				return c, false
			}
			c.message = truncate(strings.Join(msg, "\n"), maxMessageLen)
			break
		}
	}
	if c.message == "" {
		return c, false
	}
	for ; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "goroutine ") && strings.HasSuffix(lines[i], "]:") {
			break
		}
	}
	for i++; i < len(lines) && lines[i] != "" && len(c.frames) < maxFrames; i++ {
		l := lines[i]
		if strings.HasPrefix(l, "\t") || strings.HasPrefix(l, "created by ") || strings.HasPrefix(l, "...") {
			continue
		}
		c.frames = append(c.frames, MapFrame(stripArgs(l)))
	}
	return c, len(c.frames) > 0
}

// stripArgs removes the argument list "(...)" at the end of a traceback
// function line.
func stripArgs(l string) string {
	if strings.HasSuffix(l, ")") {
		if i := strings.LastIndexByte(l, '('); i > 0 {
			return l[:i]
		}
	}
	return l
}
