// Package e2e is frostyard/std's end-to-end suite. std is a library and ships
// no binary, so the closest thing it has to a program under test is the set of
// runnable examples in _examples/: each is built and run as a real subprocess
// in every output mode, and its JSON Lines stream is decoded with the
// reporter package's own ProgressEvent type. The binaries carry Go coverage
// instrumentation and write subprocess counters into a project-local
// directory; the suite fails unless those counters show reporter statements
// executed. `go test ./...` skips _examples/ (underscore directories are
// invisible to package patterns), so without this suite the examples could
// rot unnoticed.
package e2e

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/frostyard/std/reporter"
)

const reporterImportPath = "github.com/frostyard/std/reporter"

// moduleRoot returns the repository root (two levels above this file).
func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	root, err := filepath.Abs(filepath.Join(filepath.Dir(file), "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// projectTempDir creates test scratch space below tests/e2e instead of the
// system temporary directory. Subprocess coverage must remain inside the
// checkout so the harness never depends on host-global /tmp state.
func projectTempDir(t *testing.T, root, pattern string) string {
	t.Helper()
	dir, err := os.MkdirTemp(filepath.Join(root, "tests", "e2e"), pattern)
	if err != nil {
		t.Fatalf("create project-local test directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Errorf("remove project-local test directory %s: %v", dir, err)
		}
	})
	return dir
}

// exampleNames lists every runnable program under _examples/.
func exampleNames(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "_examples"))
	if err != nil {
		t.Fatalf("read _examples: %v", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		t.Fatal("no example programs found under _examples/")
	}
	return names
}

// buildExample compiles one example into dir and returns the binary path.
func buildExample(t *testing.T, root, dir, name string) string {
	t.Helper()
	bin := filepath.Join(dir, name)
	exampleImportPath := "github.com/frostyard/std/_examples/" + name
	cmd := exec.Command(
		"go", "build",
		"-cover",
		"-covermode=atomic",
		"-coverpkg="+reporterImportPath+","+exampleImportPath,
		"-o", bin,
		"./_examples/"+name,
	)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build ./_examples/%s: %v\n%s", name, err, out)
	}
	return bin
}

// run executes bin with args and returns stdout, stderr, and the exit code.
func run(t *testing.T, bin, coverageDir string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), "GOCOVERDIR="+coverageDir)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	code = 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run %s %v: %v", bin, args, err)
		}
		code = exitErr.ExitCode()
	}
	return outBuf.String(), errBuf.String(), code
}

func TestExamples(t *testing.T) {
	root := moduleRoot(t)
	names := exampleNames(t, root)
	binDir := projectTempDir(t, root, ".binaries-*")
	coverageDir := projectTempDir(t, root, ".coverage-*")
	t.Cleanup(func() {
		percent, err := reporterCoveragePercent(root, coverageDir)
		if err != nil {
			t.Errorf("verify instrumented example coverage: %v", err)
			return
		}
		t.Logf("instrumented example subprocess reporter coverage: %.1f%%", percent)
	})

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			bin := buildExample(t, root, binDir, name)

			t.Run("json", func(t *testing.T) {
				t.Parallel()
				stdout, stderr, code := run(t, bin, coverageDir, "-format", "json")
				if code != 0 {
					t.Fatalf("exit %d, stderr: %s", code, stderr)
				}
				if stderr != "" {
					t.Errorf("unexpected stderr in json mode: %q", stderr)
				}
				events := decodeEvents(t, stdout)
				if len(events) == 0 {
					t.Fatal("no events emitted")
				}
				for i, ev := range events {
					if !validEventType(ev.Type) {
						t.Errorf("event %d: unknown type %q", i, ev.Type)
					}
					if _, err := time.Parse(time.RFC3339, ev.Timestamp); err != nil {
						t.Errorf("event %d: timestamp %q is not RFC3339: %v", i, ev.Timestamp, err)
					}
				}
				if last := events[len(events)-1]; last.Type != reporter.EventTypeComplete {
					t.Errorf("last event type = %q, want %q", last.Type, reporter.EventTypeComplete)
				}
			})

			t.Run("text", func(t *testing.T) {
				t.Parallel()
				stdout, stderr, code := run(t, bin, coverageDir, "-format", "text")
				if code != 0 {
					t.Fatalf("exit %d, stderr: %s", code, stderr)
				}
				if strings.TrimSpace(stdout) == "" {
					t.Error("text mode produced no output")
				}
				// A text stream must not be mistaken for JSON Lines.
				if json.Valid([]byte(strings.SplitN(stdout, "\n", 2)[0])) {
					t.Errorf("text mode first line parses as JSON: %q", strings.SplitN(stdout, "\n", 2)[0])
				}
			})

			t.Run("noop", func(t *testing.T) {
				t.Parallel()
				stdout, stderr, code := run(t, bin, coverageDir, "-format", "noop")
				if code != 0 {
					t.Fatalf("exit %d, stderr: %s", code, stderr)
				}
				if stderr != "" {
					t.Errorf("noop mode wrote to stderr: %q", stderr)
				}
				// NoopReporter discards every event. An example may still print
				// its own non-reporter text (IsJSON() is false in noop mode, so
				// human-only tips are allowed), so compare against text mode:
				// the reporter's contribution must be gone.
				textOut, _, _ := run(t, bin, coverageDir, "-format", "text")
				if len(stdout) >= len(textOut) {
					t.Errorf("noop output (%d bytes) is not shorter than text output (%d bytes)", len(stdout), len(textOut))
				}
				for i, line := range strings.Split(stdout, "\n") {
					if json.Valid([]byte(line)) && strings.HasPrefix(line, "{") {
						t.Errorf("noop line %d is a JSON event: %s", i+1, line)
					}
				}
			})

			t.Run("unknown-format", func(t *testing.T) {
				t.Parallel()
				_, stderr, code := run(t, bin, coverageDir, "-format", "bogus")
				if code != 1 {
					t.Errorf("exit = %d, want 1", code)
				}
				if !strings.Contains(stderr, "unknown format") {
					t.Errorf("stderr = %q, want it to name the unknown format", stderr)
				}
			})
		})
	}
}

// reporterCoveragePercent reads the covdata emitted by instrumented example
// subprocesses. An empty directory, missing GOCOVERDIR wiring, or examples
// that never execute reporter code all return an error.
func reporterCoveragePercent(root, coverageDir string) (float64, error) {
	cmd := exec.Command(
		"go", "tool", "covdata", "percent",
		"-i="+coverageDir,
		"-pkg="+reporterImportPath,
	)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("read reporter coverage data: %w: %s", err, strings.TrimSpace(string(output)))
	}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] != reporterImportPath || fields[1] != "coverage:" {
			continue
		}
		percent, err := strconv.ParseFloat(strings.TrimSuffix(fields[2], "%"), 64)
		if err != nil {
			return 0, fmt.Errorf("parse reporter coverage percentage %q: %w", fields[2], err)
		}
		if percent <= 0 {
			return 0, fmt.Errorf("reporter coverage is %.1f%%; example subprocesses produced no covered statements", percent)
		}
		return percent, nil
	}
	return 0, fmt.Errorf("no reporter coverage counters found in %s: %s", coverageDir, strings.TrimSpace(string(output)))
}

func TestReporterCoveragePercentRejectsEmptyDirectory(t *testing.T) {
	root := moduleRoot(t)
	emptyDir := projectTempDir(t, root, ".empty-coverage-*")
	_, err := reporterCoveragePercent(root, emptyDir)
	if err == nil {
		t.Fatal("reporterCoveragePercent() accepted an empty coverage directory")
	}
	if !strings.Contains(err.Error(), "no reporter coverage counters") {
		t.Fatalf("reporterCoveragePercent() error = %q, want missing-counter error", err)
	}
}

// decodeEvents parses a JSON Lines stream with the reporter's own event type,
// failing on any line that is not exactly one ProgressEvent object.
func decodeEvents(t *testing.T, stream string) []reporter.ProgressEvent {
	t.Helper()
	var events []reporter.ProgressEvent
	scanner := bufio.NewScanner(strings.NewReader(stream))
	line := 0
	for scanner.Scan() {
		line++
		raw := scanner.Bytes()
		if len(bytes.TrimSpace(raw)) == 0 {
			t.Errorf("line %d: blank line inside JSON Lines stream", line)
			continue
		}
		ev, err := decodeEventLine(raw)
		if err != nil {
			t.Errorf("line %d: %v: %s", line, err, raw)
			continue
		}
		events = append(events, ev)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return events
}

func decodeEventLine(raw []byte) (reporter.ProgressEvent, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var ev reporter.ProgressEvent
	if err := dec.Decode(&ev); err != nil {
		return reporter.ProgressEvent{}, err
	}
	var trailing any
	switch err := dec.Decode(&trailing); err {
	case io.EOF:
		return ev, nil
	case nil:
		return reporter.ProgressEvent{}, errors.New("multiple JSON values on one JSON Lines line")
	default:
		return reporter.ProgressEvent{}, fmt.Errorf("trailing content after ProgressEvent: %w", err)
	}
}

func TestDecodeEventLine(t *testing.T) {
	valid := `{"type":"message","timestamp":"2026-08-19T12:00:00Z","message":"ready"}`
	another := `{"type":"complete","timestamp":"2026-08-19T12:00:01Z"}`
	tests := []struct {
		name        string
		line        string
		wantErr     bool
		errContains string
	}{
		{name: "single event", line: valid},
		{name: "single event with whitespace", line: valid + " \t"},
		{name: "concatenated second event", line: valid + another, wantErr: true, errContains: "multiple JSON values"},
		{name: "space-separated second event", line: valid + " " + another, wantErr: true, errContains: "multiple JSON values"},
		{name: "trailing junk", line: valid + " not-json", wantErr: true, errContains: "trailing content"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := decodeEventLine([]byte(tt.line))
			if tt.wantErr {
				if err == nil {
					t.Fatal("decodeEventLine() error = nil, want rejection")
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("decodeEventLine() error = %q, want substring %q", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("decodeEventLine() error = %v", err)
			}
			if event.Type != reporter.EventTypeMessage || event.Message != "ready" {
				t.Errorf("decodeEventLine() event = %+v, want message event", event)
			}
		})
	}
}

func validEventType(et reporter.EventType) bool {
	switch et {
	case reporter.EventTypeStep, reporter.EventTypeProgress, reporter.EventTypeMessage,
		reporter.EventTypeWarning, reporter.EventTypeError, reporter.EventTypeComplete:
		return true
	}
	return false
}
