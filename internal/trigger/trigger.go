package trigger

import (
	"context"
	"os/exec"
)

type Runner interface {
	RestartTarget(ctx context.Context, target string) error
}

type SystemctlRunner struct{}

func (SystemctlRunner) RestartTarget(ctx context.Context, target string) error {
	cmd := exec.CommandContext(ctx, "systemctl", "restart", "--no-block", target)
	return cmd.Run()
}
