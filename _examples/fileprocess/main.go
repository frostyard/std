// Command fileprocess simulates batch file processing to demonstrate
// the reporter package's Step, Progress, Message, and Complete methods.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/frostyard/std/reporter"
)

func main() {
	format := flag.String("format", "text", "output format: text, json, noop")
	flag.Parse()

	var r reporter.Reporter
	switch *format {
	case "text":
		r = reporter.NewTextReporter(os.Stdout)
	case "json":
		r = reporter.NewJSONReporter(os.Stdout)
	case "noop":
		r = reporter.NoopReporter{}
	default:
		fmt.Fprintf(os.Stderr, "unknown format: %s\n", *format)
		os.Exit(1)
	}

	files := []struct {
		name  string
		pages int
		skip  bool
	}{
		{"document.pdf", 42, false},
		{"photo.jpg", 1, false},
		{"archive.tar.gz", 0, true},
		{"report.docx", 18, false},
		{"notes.txt", 3, true},
	}

	succeeded := 0
	skipped := 0

	for i, f := range files {
		r.Step(i+1, len(files), fmt.Sprintf("Scanning %s", f.name))
		time.Sleep(200 * time.Millisecond)

		if f.skip {
			r.Message("Skipped (unsupported format)")
			skipped++
			continue
		}

		r.Message("Found %d pages", f.pages)
		r.Progress((i+1)*100/len(files), fmt.Sprintf("%d%% complete", (i+1)*100/len(files)))
		succeeded++
	}

	r.Complete(
		fmt.Sprintf("Processed %d files (%d succeeded, %d skipped)", len(files), succeeded, skipped),
		nil,
	)

	if !r.IsJSON() {
		fmt.Println("\nTip: run with --format=json for machine-readable output")
	}
}
