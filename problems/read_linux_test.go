package problems

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeJournalctl writes a shell script that saves its arguments and prints
// output, or fails.
func fakeJournalctl(t *testing.T, dir, output string, fail bool) (bin, argsFile string) {
	t.Helper()
	argsFile = filepath.Join(dir, "args")
	outFile := filepath.Join(dir, "out")
	writeFile(t, outFile, output)
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > '" + argsFile + "'\n"
	if fail {
		script += "echo 'Failed to open the journal: permission denied' >&2\nexit 1\n"
	} else {
		script += "cat '" + outFile + "'\n"
	}
	bin = filepath.Join(dir, "journalctl")
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return bin, argsFile
}

func readArgs(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(strings.ReplaceAll(string(b), "\n", " "))
}

func TestRead(t *testing.T) {
	dir := t.TempDir()
	out := jsonLine(t, goErrorEntry()) + "\n" + jsonLine(t, map[string]any{"MESSAGE_ID": MessageID}) + "\n"
	bin, argsFile := fakeJournalctl(t, dir, out, false)

	res, err := Read(context.Background(), ReadOptions{
		User: true, Since: time.Unix(1_800_000_000, 0), Limit: 5, Journalctl: bin,
	})
	if err != nil {
		t.Fatalf("Read = %v", err)
	}
	if len(res.Records) != 1 || res.Dropped != 1 {
		t.Errorf("Read = %+v, want 1 record and 1 dropped", res)
	}
	want := "--no-pager -q -o json --all --user --since=@1800000000 -n 5 MESSAGE_ID=" + MessageID
	if got := readArgs(t, argsFile); got != want {
		t.Errorf("args = %q, want %q", got, want)
	}

	// The defaults: the system journal, journalctl from PATH.
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if _, err := Read(context.Background(), ReadOptions{}); err != nil {
		t.Fatalf("Read = %v", err)
	}
	want = "--no-pager -q -o json --all --system MESSAGE_ID=" + MessageID
	if got := readArgs(t, argsFile); got != want {
		t.Errorf("args = %q, want %q", got, want)
	}
}

func TestReadFails(t *testing.T) {
	bin, _ := fakeJournalctl(t, t.TempDir(), "", true)
	_, err := Read(context.Background(), ReadOptions{Journalctl: bin})
	if err == nil || !strings.Contains(err.Error(), "permission denied") || !strings.Contains(err.Error(), "exit status 1") {
		t.Errorf("Read = %v, want the exit status and stderr", err)
	}
	_, err = Read(context.Background(), ReadOptions{Journalctl: filepath.Join(t.TempDir(), "missing")})
	if err == nil {
		t.Error("Read with a missing journalctl: want an error")
	}
}
