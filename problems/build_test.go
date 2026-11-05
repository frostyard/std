package problems

import (
	"os"
	"os/exec"
	"testing"
)

// TestBuildOtherOS checks that the package builds where there is no
// systemd journal.
func TestBuildOtherOS(t *testing.T) {
	if testing.Short() {
		t.Skip("cross build in -short mode")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go command")
	}
	for _, goos := range []string{"windows", "darwin"} {
		cmd := exec.Command(goBin, "build", ".")
		cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH=amd64", "CGO_ENABLED=0")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("GOOS=%s go build: %v\n%s", goos, err, out)
		}
	}
}
