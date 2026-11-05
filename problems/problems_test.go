package problems

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// TestMain runs a helper program instead of the tests when
// PROBLEMS_TEST_HELPER is set: the subprocess tests re-run the test binary.
func TestMain(m *testing.M) {
	switch os.Getenv("PROBLEMS_TEST_HELPER") {
	case "":
		os.Exit(m.Run())
	case "recover":
		helperRecover(false)
	case "recover-crash":
		helperRecover(true)
	case "recover-crash-multiline":
		helperRecoverMultiline()
	case "crash":
		helperCrash()
	case "leftover":
		helperLeftover()
	}
	os.Exit(3)
}

func helperRecorder() *Recorder {
	version := os.Getenv("PROBLEMS_TEST_VERSION")
	if version == "" {
		version = "1.2.3"
	}
	return New("org.frostyard.nsl", version, ComponentCLI, WithSocket(os.Getenv("PROBLEMS_TEST_SOCKET")))
}

func helperRecover(crashOutput bool) {
	r := helperRecorder()
	if crashOutput {
		if err := r.CrashOutput(); err != nil {
			os.Exit(4)
		}
	}
	defer r.RecoverPanic()
	panickingHelper()
}

func helperRecoverMultiline() {
	r := helperRecorder()
	if err := r.CrashOutput(); err != nil {
		os.Exit(4)
	}
	defer r.RecoverPanic()
	panic("first\nsecond")
}

//go:noinline
func panickingHelper() {
	panic("helper boom")
}

func helperCrash() {
	r := helperRecorder()
	if err := r.CrashOutput(); err != nil {
		os.Exit(4)
	}
	go crashingGoroutine()
	select {}
}

//go:noinline
func crashingGoroutine() {
	panic("crash boom")
}

func helperLeftover() {
	if err := helperRecorder().CrashOutput(); err != nil {
		os.Exit(4)
	}
	os.Exit(0)
}

func TestMapFrame(t *testing.T) {
	tests := map[string]string{
		"github.com/frostyard/std/problems.(*Recorder).RecoverPanic": "github.com/frostyard/std/problems.(*Recorder).RecoverPanic",
		"runtime.gopanic":                  "runtime.gopanic",
		"main.main.func1":                  "main.main.func1",
		"github.com/spf13/cobra.Execute":   "other",
		"/home/u/main.go:12":               "other",
		"main.f(0x1, 0x2)":                 "other",
		"panic":                            "other",
		"":                                 "other",
		"main." + strings.Repeat("a", 155): "main." + strings.Repeat("a", 155),
		"main." + strings.Repeat("a", 156): "other",
	}
	for in, want := range tests {
		if got := MapFrame(in); got != want {
			t.Errorf("MapFrame(%q) = %q, want %q", in, got, want)
		}
	}
}

func validFields() map[string]string {
	return map[string]string{
		"MESSAGE_ID":            MessageID,
		"MESSAGE":               "text",
		"PRIORITY":              "3",
		"_UID":                  "1000",
		"FROSTYARD_SCHEMA":      Schema,
		"FROSTYARD_APP_ID":      "org.frostyard.nsl",
		"FROSTYARD_APP_VERSION": "1.2.3",
		"FROSTYARD_RUNTIME":     "go",
		"FROSTYARD_COMPONENT":   "cli",
		"FROSTYARD_CLASS":       "error",
		"FROSTYARD_CODE":        "a.b",
	}
}

func TestValidate(t *testing.T) {
	d, err := Validate(validFields())
	if err != nil {
		t.Fatalf("Validate(valid) = %v", err)
	}
	want := Detail{AppID: "org.frostyard.nsl", AppVersion: "1.2.3", Runtime: "go", Component: "cli", Class: "error", Code: "a.b"}
	if !reflect.DeepEqual(d, want) {
		t.Errorf("Validate = %+v, want %+v", d, want)
	}

	// The vectors cover each rule; this checks that every error wraps
	// ErrInvalid.
	f := validFields()
	f["FROSTYARD_APP_ID"] = "org.example"
	if _, err := Validate(f); !errors.Is(err, ErrInvalid) {
		t.Errorf("Validate(bad app id) = %v, want ErrInvalid", err)
	}
}

func TestDetailJSON(t *testing.T) {
	b, err := json.Marshal(Detail{AppID: "org.frostyard.nsl", AppVersion: "1.2.3", Runtime: "go", Component: "cli", Class: "error", Code: "x"})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"app_id":"org.frostyard.nsl","app_version":"1.2.3","runtime":"go","component":"cli","class":"error","code":"x"}`
	if string(b) != want {
		t.Errorf("json = %s, want %s", b, want)
	}
}

// vector is 1 block of testdata/vectors/v1.txt.
type vector struct {
	name    string
	kv      map[string][]string
	journal []field // in order; FROSTYARD_FRAMES lines joined
}

func (v vector) get(key string) string {
	if l := v.kv[key]; len(l) > 0 {
		return l[0]
	}
	return ""
}

func loadVectors(t *testing.T) []vector {
	t.Helper()
	f, err := os.Open("testdata/vectors/v1.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []vector
	cur := vector{kv: map[string][]string{}}
	flush := func() {
		if len(cur.kv) > 0 {
			if cur.name == "" {
				t.Fatalf("vector without a name: %v", cur.kv)
			}
			out = append(out, cur)
		}
		cur = vector{kv: map[string][]string{}}
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == "":
			flush()
			continue
		case strings.HasPrefix(line, "#"):
			continue
		}
		key, value, ok := strings.Cut(line, ": ")
		if !ok {
			key, ok = strings.CutSuffix(line, ":")
			if !ok {
				t.Fatalf("bad vector line %q", line)
			}
		}
		cur.kv[key] = append(cur.kv[key], value)
		if key == "vector" {
			cur.name = value
		}
		if name, ok := strings.CutPrefix(key, "journal."); ok {
			if n := len(cur.journal); n > 0 && cur.journal[n-1].name == name && name == fieldFrames {
				cur.journal[n-1].value += "\n" + value
			} else {
				cur.journal = append(cur.journal, field{name, value})
			}
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	flush()
	return out
}

func journalMap(fs []field) map[string]string {
	m := map[string]string{}
	for _, f := range fs {
		m[f.name] = f.value
	}
	return m
}

func payloadDetail(v vector) Detail {
	return Detail{
		AppID:      v.get("payload.app_id"),
		AppVersion: v.get("payload.app_version"),
		Runtime:    v.get("payload.runtime"),
		Component:  v.get("payload.component"),
		Class:      v.get("payload.class"),
		Code:       v.get("payload.code"),
		Frames:     v.kv["payload.frames"],
		Location:   v.get("payload.location"),
	}
}

func TestVectors(t *testing.T) {
	vectors := loadVectors(t)
	if len(vectors) < 40 {
		t.Fatalf("loaded %d vectors, want at least 40", len(vectors))
	}
	counts := map[string]int{}
	seen := map[string]bool{}
	for _, v := range vectors {
		if seen[v.name] {
			t.Errorf("vector %s: name repeats", v.name)
		}
		seen[v.name] = true
		typ := v.get("type")
		counts[typ]++
		t.Run(v.name, func(t *testing.T) {
			switch typ {
			case "frame":
				if got, want := MapFrame(v.get("in.frame")), v.get("out.frame"); got != want {
					t.Errorf("MapFrame(%q) = %q, want %q", v.get("in.frame"), got, want)
				}
			case "write":
				// The writer transport exists only on Linux; the
				// validation below runs on every OS.
				if v.get("in.runtime") == "go" && runtime.GOOS == "linux" {
					runWriteVector(t, v)
				}
				if v.get("expect") == "ok" {
					checkValidate(t, v)
				}
			case "validate":
				checkValidate(t, v)
			default:
				t.Fatalf("unknown type %q", typ)
			}
		})
	}
	for _, typ := range []string{"write", "validate", "frame"} {
		if counts[typ] == 0 {
			t.Errorf("no %s vectors", typ)
		}
	}
}

func checkValidate(t *testing.T, v vector) {
	t.Helper()
	d, err := Validate(journalMap(v.journal))
	switch v.get("expect") {
	case "ok":
		if err != nil {
			t.Fatalf("Validate = %v, want ok", err)
		}
		if want := payloadDetail(v); !reflect.DeepEqual(d, want) {
			t.Errorf("Validate = %+v, want %+v", d, want)
		}
	case "reject":
		if err == nil {
			t.Errorf("Validate accepted a record that the vector rejects: %+v", d)
		}
	default:
		t.Fatalf("bad expect %q", v.get("expect"))
	}
}
