package problems

import (
	"slices"
	"strings"
)

// Crash file format: the header line, then whatever the Go runtime writes
// through runtime/debug.SetCrashOutput (a traceback), and maybe the marker.
const (
	crashHeaderVersion = "frostyard-problems/1"
	recordedMarker     = "frostyard-problems: recorded"
	crashReadMax       = 64 << 10
	crashMaxPerStart   = 8
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
// whose first panic line ends with " [recovered, repanicked]"; a fatal error
// or a plain panic gives a record even when the marker is there, because
// other code (for example net/http) can recover the re-panic and the
// process can crash later.
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
		if strings.HasPrefix(l, "panic: ") || strings.HasPrefix(l, "fatal error: ") {
			if strings.HasPrefix(l, "panic: ") && strings.HasSuffix(l, repanicSuffix) && slices.Contains(lines, recordedMarker) {
				return c, false
			}
			c.message = truncate(l, maxMessageLen)
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
