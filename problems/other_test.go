//go:build !linux

package problems

import (
	"context"
	"errors"
	"testing"
)

func TestOtherOS(t *testing.T) {
	if _, err := Read(context.Background(), ReadOptions{}); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("Read = %v, want ErrUnsupported", err)
	}
	r := New("org.frostyard.nsl", "1.2.3", ComponentCLI)
	if err := r.CrashOutput(); err != nil {
		t.Errorf("CrashOutput = %v", err)
	}
	if st := r.recordError("x", nil); st != statusDroppedOther {
		t.Errorf("status = %v, want dropped-other", st)
	}
	var got any
	func() {
		defer func() { got = recover() }()
		defer r.RecoverPanic()
		panic("still panics")
	}()
	if got != "still panics" {
		t.Errorf("re-panic value = %v", got)
	}
}
