//go:build !linux

package problems

import (
	"context"
	"errors"
	"fmt"
)

// sendDatagram does nothing on this OS: there is no systemd journal.
func sendDatagram(string, []byte) status { return statusDroppedOther }

// markRecorded does nothing on this OS: CrashOutput is not active.
func markRecorded() {}

// CrashOutput does nothing on this OS. On an inert Recorder it returns Err.
func (r *Recorder) CrashOutput() error {
	return r.Err()
}

// Read is not supported on this OS: it returns an error that wraps
// errors.ErrUnsupported.
func Read(context.Context, ReadOptions) (Result, error) {
	return Result{}, fmt.Errorf("problems: read the journal: %w", errors.ErrUnsupported)
}
