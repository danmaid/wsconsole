//go:build linux
// +build linux

package systemd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
)

// RunLoginPTY spawns a login shell using the specified strategy.
// If strategy is empty, auto-detection is performed.
// This is the main entry point for launching a login PTY.
//
// Returns the command, PTY master file descriptor, cleanup function, and error.
func RunLoginPTY(ctx context.Context, strategy LoginStrategy) (cmd *exec.Cmd, ptyMaster *os.File, cleanup func() error, err error) {
	// Create a PTY master/slave pair
	master, slave, err := openPTY()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to open PTY: %w", err)
	}

	// Select launcher strategy
	launcher, err := SelectLauncher(strategy)
	if err != nil {
		if err := master.Close(); err != nil {

			slog.Warn("failed to close master", "error", err)

		}
		if err := slave.Close(); err != nil {
			slog.Warn("failed to close slave", "error", err)
		}
		return nil, nil, nil, fmt.Errorf("failed to select launcher: %w", err)
	}

	slog.Debug("using launcher strategy", "strategy", launcher.Name())

	// Launch the login process
	cmd, err = launcher.Launch(ctx, slave)
	if err != nil {
		if err := master.Close(); err != nil {

			slog.Warn("failed to close master", "error", err)

		}
		if err := slave.Close(); err != nil {
			slog.Warn("failed to close slave", "error", err)
		}
		return nil, nil, nil, fmt.Errorf("failed to launch login: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		if err := master.Close(); err != nil {

			slog.Warn("failed to close master", "error", err)

		}
		if err := slave.Close(); err != nil {
			slog.Warn("failed to close slave", "error", err)
		}
		return nil, nil, nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Close slave in parent process (child still has it)
	if err := slave.Close(); err != nil {

		slog.Warn("failed to close slave", "error", err)

	}

	slog.Info("PTY shell started", "pid", cmd.Process.Pid, "launcher", launcher.Name())

	cleanup = func() error {
		if master != nil {
			if err := master.Close(); err != nil {
				return fmt.Errorf("failed to close PTY master: %w", err)
			}
		}
		return nil
	}

	return cmd, master, cleanup, nil
}
