package reporter

import (
	"errors"
	"testing"
)

func TestNoopReporter(t *testing.T) {
	r := NoopReporter{}

	r.Step(1, 3, "test")
	r.Progress(50, "test")
	r.Message("hello %s", "world")
	r.MessagePlain("hello %s", "world")
	r.Warning("careful %s", "now")
	r.Error(errors.New("boom"), "oops")
	r.Complete("done", nil)

	if r.IsJSON() {
		t.Error("NoopReporter.IsJSON() = true, want false")
	}
}
