// Package e2e is frostyard/std's end-to-end suite. std is a library and ships
// no binary, so the closest thing it has to a program under test is the set of
// runnable examples in _examples/: each is built and run as a real subprocess
// in every output mode, and its JSON Lines stream is decoded with the
// reporter package's own ProgressEvent type. `go test ./...` skips _examples/
// (underscore directories are invisible to package patterns), so without this
// suite the examples could rot unnoticed.
package e2e

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/frostyard/std/reporter"
)

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
	cmd := exec.Command("go", "build", "-o", bin, "./_examples/"+name)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build ./_examples/%s: %v\n%s", name, err, out)
	}
	return bin
}

// run executes bin with args and returns stdout, stderr, and the exit code.
func run(t *testing.T, bin string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
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
	binDir := t.TempDir()

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			bin := buildExample(t, root, binDir, name)

			t.Run("json", func(t *testing.T) {
				t.Parallel()
				stdout, stderr, code := run(t, bin, "-format", "json")
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
				stdout, stderr, code := run(t, bin, "-format", "text")
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
				stdout, stderr, code := run(t, bin, "-format", "noop")
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
				textOut, _, _ := run(t, bin, "-format", "text")
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
				_, stderr, code := run(t, bin, "-format", "bogus")
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
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		var ev reporter.ProgressEvent
		if err := dec.Decode(&ev); err != nil {
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

func validEventType(et reporter.EventType) bool {
	switch et {
	case reporter.EventTypeStep, reporter.EventTypeProgress, reporter.EventTypeMessage,
		reporter.EventTypeWarning, reporter.EventTypeError, reporter.EventTypeComplete:
		return true
	}
	return false
}
