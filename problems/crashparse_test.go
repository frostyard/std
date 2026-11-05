package problems

import (
	"reflect"
	"strings"
	"testing"
)

const crashFixtureHeader = "frostyard-problems/1 app=org.frostyard.nsl version=1.2.3 component=daemon\n"

const crashFixturePanic = crashFixtureHeader + `panic: boom

goroutine 7 [running]:
panic({0x4b2a40?, 0x5153d0?})
	/usr/lib/go/src/runtime/panic.go:787 +0x132
github.com/frostyard/nsl/internal/store.(*DB).Open(0xc000123456, {0x5a1b2c, 0x4})
	/home/brian/src/nsl/internal/store/db.go:42 +0x1d
github.com/spf13/cobra.(*Command).execute(0xc0000a2000, {0xc000010050, 0x1, 0x1})
	/home/brian/go/pkg/mod/github.com/spf13/cobra@v1.8.0/command.go:983 +0xabc
main.run.func1(...)
	/home/brian/src/nsl/main.go:20
...additional frames elided...
created by main.run in goroutine 1
	/home/brian/src/nsl/main.go:18 +0x1a

goroutine 1 [chan receive]:
main.main()
	/home/brian/src/nsl/main.go:10 +0x10
`

const crashFixtureFatal = crashFixtureHeader + `fatal error: concurrent map writes

goroutine 7 gp=0xc000007a40 m=4 mp=0xc000080008 [running]:
internal/runtime/maps.fatal({0x4b0e2f?, 0x0?})
	/usr/lib/go/src/runtime/panic.go:1058 +0x18
main.main.func1()
	/home/brian/src/x/main.go:9 +0x45
created by main.main in goroutine 1
	/home/brian/src/x/main.go:7 +0x2a
`

func TestParseCrash(t *testing.T) {
	c, ok := parseCrash([]byte(crashFixturePanic))
	if !ok {
		t.Fatal("parseCrash(panic) = no record")
	}
	want := crashReport{
		appID: "org.frostyard.nsl", version: "1.2.3", component: ComponentDaemon,
		message: "panic: boom",
		frames:  []string{"other", "github.com/frostyard/nsl/internal/store.(*DB).Open", "other", "main.run.func1"},
	}
	if !reflect.DeepEqual(c, want) {
		t.Errorf("parseCrash = %+v, want %+v", c, want)
	}
	checkFrames(t, c.frames)

	c, ok = parseCrash([]byte(crashFixtureFatal))
	if !ok {
		t.Fatal("parseCrash(fatal) = no record")
	}
	if c.message != "fatal error: concurrent map writes" ||
		!reflect.DeepEqual(c.frames, []string{"internal/runtime/maps.fatal", "main.main.func1"}) {
		t.Errorf("parseCrash(fatal) = %+v", c)
	}
}

func TestParseCrashFramesCap(t *testing.T) {
	var b strings.Builder
	b.WriteString(crashFixtureHeader + "panic: deep\n\ngoroutine 1 [running]:\n")
	for range 40 {
		b.WriteString("main.deep(0x1)\n\t/src/main.go:5 +0x1\n")
	}
	c, ok := parseCrash([]byte(b.String()))
	if !ok || len(c.frames) != maxFrames {
		t.Errorf("parseCrash = %d frames, %v; want %d", len(c.frames), ok, maxFrames)
	}
}

func TestParseCrashNoRecord(t *testing.T) {
	tests := map[string]string{
		"empty":           "",
		"header only":     crashFixtureHeader,
		"no header":       strings.TrimPrefix(crashFixturePanic, crashFixtureHeader),
		"other version":   strings.Replace(crashFixturePanic, "frostyard-problems/1", "frostyard-problems/2", 1),
		"short header":    "frostyard-problems/1 app=org.frostyard.nsl version=1.2.3\npanic: x\n\ngoroutine 1 [running]:\nmain.main()\n",
		"bad header keys": strings.Replace(crashFixturePanic, "app=", "id=", 1),
		"no goroutine":    crashFixtureHeader + "panic: x\n",
		"no frames":       crashFixtureHeader + "panic: x\n\ngoroutine 1 [running]:\n\n",
	}
	for name, in := range tests {
		if c, ok := parseCrash([]byte(in)); ok {
			t.Errorf("%s: parseCrash = %+v, want no record", name, c)
		}
	}
}

// crashFixtureRepanic is the crash file of a panic that RecoverPanic
// recorded: the marker, then the re-panic report.
const crashFixtureRepanic = crashFixtureHeader + recordedMarker + `
panic: boom [recovered, repanicked]

goroutine 1 [running]:
github.com/frostyard/std/problems.(*Recorder).RecoverPanic(0xc000010000)
	/src/problems/recorder.go:161 +0xcc
panic({0x5ee380?, 0x64c0d0?})
	/usr/lib/go/src/runtime/panic.go:860 +0x13a
main.main()
	/src/main.go:10 +0x25
`

func TestParseCrashMarker(t *testing.T) {
	marker := recordedMarker + "\n"
	tests := []struct {
		name    string
		data    string
		ok      bool
		message string
	}{
		// (a) The re-panic of a recorded panic: no second record.
		{"re-panic", crashFixtureRepanic, false, ""},
		{"re-panic marker at the end", crashFixtureHeader + strings.TrimPrefix(crashFixtureRepanic, crashFixtureHeader+marker) + marker, false, ""},
		// (b) Other code recovered the re-panic; a later plain panic of
		// another goroutine crashed the process.
		{"later plain panic", crashFixtureHeader + marker + strings.TrimPrefix(crashFixturePanic, crashFixtureHeader), true, "panic: boom"},
		// (c) A later fatal error.
		{"later fatal error", crashFixtureHeader + marker + strings.TrimPrefix(crashFixtureFatal, crashFixtureHeader), true, "fatal error: concurrent map writes"},
		// A multi-line panic value: Go 1.26 puts the suffix on the last,
		// tab-indented, line.
		{"multi-line re-panic", crashFixtureHeader + marker + "panic: first\n\tsecond [recovered, repanicked]\n\ngoroutine 1 [running]:\nmain.main()\n", false, ""},
		{"multi-line plain panic", crashFixtureHeader + marker + "panic: g1\n\tg2\n\ngoroutine 7 [running]:\nmain.main.func1()\n", true, "panic: g1\ng2"},
		{"multi-line re-panic without marker", crashFixtureHeader + "panic: first\n\tsecond [recovered, repanicked]\n\ngoroutine 1 [running]:\nmain.main()\n", true, "panic: first\nsecond [recovered, repanicked]"},
		// A re-panic with no marker: RecoverPanic did not record it.
		{"re-panic without marker", strings.Replace(crashFixtureRepanic, marker, "", 1), true, "panic: boom [recovered, repanicked]"},
	}
	for _, tt := range tests {
		c, ok := parseCrash([]byte(tt.data))
		if ok != tt.ok {
			t.Errorf("%s: parseCrash ok = %v, want %v", tt.name, ok, tt.ok)
			continue
		}
		if ok {
			if c.message != tt.message || len(c.frames) == 0 {
				t.Errorf("%s: parseCrash = %+v, want message %q and frames", tt.name, c, tt.message)
			}
			checkFrames(t, c.frames)
		}
	}
}

func TestStripArgs(t *testing.T) {
	tests := map[string]string{
		"main.main()":                    "main.main",
		"main.(*T).M(0xc000010000, 0x1)": "main.(*T).M",
		"main.F[...](...)":               "main.F[...]",
		"main.main":                      "main.main",
		"(x)":                            "(x)",
	}
	for in, want := range tests {
		if got := stripArgs(in); got != want {
			t.Errorf("stripArgs(%q) = %q, want %q", in, got, want)
		}
	}
}
