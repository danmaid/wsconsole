//go:build linux
// +build linux

package pty

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"golang.org/x/sys/unix"
)

const (
	// PTYBufferSize is the buffer size for reading from PTY (64KB chunks)
	PTYBufferSize = 64 * 1024
)

// SetWinsize sets the terminal window size for the given PTY file descriptor.
func SetWinsize(fd uintptr, cols, rows int) error {
	ws := &unix.Winsize{
		Row: uint16(rows),
		Col: uint16(cols),
	}

	if err := unix.IoctlSetWinsize(int(fd), unix.TIOCSWINSZ, ws); err != nil {
		return fmt.Errorf("failed to set window size: %w", err)
	}

	slog.Debug("window size updated", "cols", cols, "rows", rows)
	return nil
}

// GetWinsize retrieves the current terminal window size.
func GetWinsize(fd uintptr) (cols, rows int, err error) {
	ws, err := unix.IoctlGetWinsize(int(fd), unix.TIOCGWINSZ)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get window size: %w", err)
	}
	return int(ws.Col), int(ws.Row), nil
}

// SetNonBlocking sets the PTY file descriptor to non-blocking mode.
func SetNonBlocking(file *os.File) error {
	fd := int(file.Fd())
	if err := unix.SetNonblock(fd, true); err != nil {
		return fmt.Errorf("failed to set non-blocking: %w", err)
	}

	slog.Debug("PTY set to non-blocking mode", "fd", fd)
	return nil
}

// CopyPTYToWriter reads from PTY master and writes to output (e.g., WebSocket).
// Uses 64KB buffer chunks for immediate flushing.
// This function blocks until EOF or error.
func CopyPTYToWriter(ptyMaster *os.File, writer io.Writer) error {
	buf := make([]byte, PTYBufferSize)
	for {
		n, err := ptyMaster.Read(buf)
		if n > 0 {
			if _, writeErr := writer.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("failed to write to output: %w", writeErr)
			}
		}
		if err != nil {
			if err == io.EOF {
				slog.Info("PTY closed (EOF)")
				return nil
			}
			return fmt.Errorf("failed to read from PTY: %w", err)
		}
	}
}

// CopyReaderToPTY reads from input (e.g., WebSocket) and writes to PTY master.
func CopyReaderToPTY(reader io.Reader, ptyMaster *os.File) error {
	buf := make([]byte, PTYBufferSize)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			if _, writeErr := ptyMaster.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("failed to write to PTY: %w", writeErr)
			}
		}
		if err != nil {
			if err == io.EOF {
				slog.Info("input closed (EOF)")
				return nil
			}
			return fmt.Errorf("failed to read from input: %w", err)
		}
	}
}
