package executil

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

func Run(ctx context.Context, timeout time.Duration, name string, args ...string) (string, error) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

func SystemctlActive(ctx context.Context, timeout time.Duration, unit string) bool {
	_, err := Run(ctx, timeout, "systemctl", "is-active", "--quiet", unit)
	return err == nil
}
