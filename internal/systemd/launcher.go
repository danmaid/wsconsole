//go:build linux
// +build linux

package systemd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
)

// LoginStrategy defines the strategy for launching a login shell
type LoginStrategy string

const (
	StrategyAuto       LoginStrategy = "auto"        // Auto-detect the best method
	StrategyDirect     LoginStrategy = "direct"      // Direct /bin/login (requires UID=0)
	StrategySystemdRun LoginStrategy = "systemd-run" // Use systemd-run for privilege escalation
)

// LoginLauncher defines the interface for launching a login shell
type LoginLauncher interface {
	// Launch starts the login process and returns the command and PTY slave
	Launch(ctx context.Context, slave *os.File) (*exec.Cmd, error)
	// Name returns the name of this launcher strategy
	Name() string
}

// DirectLauncher directly forks /bin/login (requires UID=0)
type DirectLauncher struct{}

func (l *DirectLauncher) Name() string {
	return "direct"
}

func (l *DirectLauncher) Launch(ctx context.Context, slave *os.File) (*exec.Cmd, error) {
	if os.Getuid() != 0 {
		return nil, fmt.Errorf("direct launcher requires UID=0")
	}
	cmd := exec.CommandContext(ctx, "/bin/login")
	cmd.Stdin = slave
	cmd.Stdout = slave
	cmd.Stderr = slave
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
	}
	return cmd, nil
}

// SystemdRunLauncher uses systemd-run to escalate privileges and launch /bin/login
type SystemdRunLauncher struct{}

func (l *SystemdRunLauncher) Name() string {
	return "systemd-run"
}

func (l *SystemdRunLauncher) Launch(ctx context.Context, slave *os.File) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx,
		"systemd-run",
		"--uid=0",
		"--pty",
		"--quiet",
		"--collect",
		"--wait",
		"--service-type=exec",
		"/bin/login",
	)
	cmd.Stdin = slave
	cmd.Stdout = slave
	cmd.Stderr = slave
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
	}
	return cmd, nil
}

// SelectLauncher determines the best LoginLauncher based on environment and permissions
func SelectLauncher(strategy LoginStrategy) (LoginLauncher, error) {
	if strategy != StrategyAuto {
		// Use explicitly specified strategy
		return newLauncherByStrategy(strategy)
	}

	// Auto-detect: check UID first
	if os.Getuid() == 0 {
		slog.Info("auto-selected launcher strategy", "strategy", "direct", "reason", "UID=0")
		return &DirectLauncher{}, nil
	}

	// Not root, try systemd-run
	if _, err := exec.LookPath("systemd-run"); err == nil {
		slog.Info("auto-selected launcher strategy", "strategy", "systemd-run", "reason", "UID!=0")
		return &SystemdRunLauncher{}, nil
	}

	return nil, fmt.Errorf("cannot launch /bin/login: not running as root and systemd-run not available")
}

func newLauncherByStrategy(strategy LoginStrategy) (LoginLauncher, error) {
	switch strategy {
	case StrategyDirect:
		if os.Getuid() != 0 {
			return nil, fmt.Errorf("direct launcher requires UID=0, got UID=%d", os.Getuid())
		}
		return &DirectLauncher{}, nil

	case StrategySystemdRun:
		// Check if systemd-run is available
		if _, err := exec.LookPath("systemd-run"); err != nil {
			return nil, fmt.Errorf("systemd-run not found: %w", err)
		}
		return &SystemdRunLauncher{}, nil

	default:
		return nil, fmt.Errorf("unknown launcher strategy: %s", strategy)
	}
}

// openPTY creates a new PTY master/slave pair.
func openPTY() (master, slave *os.File, err error) {
	master, slave, err = pty.Open()
	if err != nil {
		return nil, nil, fmt.Errorf("pty.Open failed: %w", err)
	}
	return master, slave, nil
}
