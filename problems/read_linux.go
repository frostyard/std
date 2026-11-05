//go:build linux

package problems

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// maxStderr bounds the journalctl error text that Read returns.
const maxStderr = 1024

// Read runs journalctl and returns the valid problem records of the user or
// the system journal:
//
//	journalctl --no-pager -q -o json --all --user|--system [--since=@<unix>] [-n <limit>] MESSAGE_ID=<MessageID>
//
// --all keeps journalctl from printing a field longer than 4096 bytes as
// null (FROSTYARD_FRAMES can be longer). On a non-zero exit, Read returns an
// error with the start of journalctl's stderr.
func Read(ctx context.Context, opts ReadOptions) (Result, error) {
	bin := opts.Journalctl
	if bin == "" {
		bin = "journalctl"
	}
	args := []string{"--no-pager", "-q", "-o", "json", "--all"}
	if opts.User {
		args = append(args, "--user")
	} else {
		args = append(args, "--system")
	}
	if !opts.Since.IsZero() {
		args = append(args, "--since=@"+strconv.FormatInt(opts.Since.Unix(), 10))
	}
	if opts.Limit > 0 {
		args = append(args, "-n", strconv.Itoa(opts.Limit))
	}
	args = append(args, "MESSAGE_ID="+MessageID)

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(truncate(stderr.String(), maxStderr))
		return Result{}, fmt.Errorf("problems: journalctl: %w: %s", err, msg)
	}
	return ParseJSON(&stdout)
}
