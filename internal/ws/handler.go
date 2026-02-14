//go:build linux
// +build linux

package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/danmaid/wsconsole/internal/pty"
	"github.com/danmaid/wsconsole/internal/systemd"
	"github.com/gorilla/websocket"
)

// Message represents the WebSocket JSON message protocol (optional mode).
type Message struct {
	Type string `json:"type"`           // "resize"
	Cols int    `json:"cols,omitempty"` // for "resize" type
	Rows int    `json:"rows,omitempty"` // for "resize" type
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 30 * time.Second // 30 seconds for ping/pong
	pingPeriod     = (pongWait * 9) / 10
	idleTimeout    = 5 * time.Minute // 5 minutes idle disconnect
	ptyBufferSize  = 64 * 1024       // 64KB chunks for PTY reads
	maxMessageSize = 512 * 1024      // 512KB max message size
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  64 * 1024,
	WriteBufferSize: 64 * 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for simplicity (adjust for production)
		return true
	},
}

// Handler handles WebSocket connections and bridges PTY I/O.
// Supports both binary transparent mode (default) and JSON message mode.
func Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("failed to upgrade WebSocket", "error", err)
		return
	}
	defer func() {
		if err := conn.Close(); err != nil {
			slog.Warn("failed to close connection", "error", err)
		}
	}()

	slog.Info("WebSocket connection established", "remote", r.RemoteAddr)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Determine login launcher strategy from query parameter
	strategy := systemd.StrategyAuto
	if strategyParam := r.URL.Query().Get("launcher"); strategyParam != "" {
		strategy = systemd.LoginStrategy(strategyParam)
	}

	// Start login shell with selected launcher strategy
	cmd, ptyMaster, cleanup, err := systemd.RunLoginPTY(ctx, strategy)
	if err != nil {
		slog.Error("failed to start login PTY", "error", err, "strategy", strategy)
		sendCloseMessage(conn, websocket.CloseInternalServerErr, fmt.Sprintf("Failed to start login: %v", err))
		return
	}
	defer func() {
		if cmd.Process != nil {
			slog.Debug("killing process", "pid", cmd.Process.Pid)
			if err := cmd.Process.Kill(); err != nil {
				slog.Warn("failed to kill process", "error", err)
			}
		}
		if err := cleanup(); err != nil {
			slog.Warn("failed to cleanup", "error", err)
		}
		if err := cmd.Wait(); err != nil {
			slog.Warn("failed to wait for process", "error", err)
		}
	}()

	// Determine mode: check query parameter ?mode=json for JSON mode
	mode := r.URL.Query().Get("mode")
	useBinaryMode := mode != "json"

	// Setup ping/pong with idle timeout
	if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		slog.Warn("failed to set read deadline", "error", err)
	}
	conn.SetPongHandler(func(string) error {
		if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			slog.Warn("failed to set read deadline", "error", err)
		}
		return nil
	})

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine 1: Read from PTY, send to WebSocket
	go func() {
		defer wg.Done()
		defer cancel()
		if err := ptyToWebSocket(conn, ptyMaster, useBinaryMode); err != nil {
			if err != io.EOF {
				slog.Error("PTY to WebSocket error", "error", err)
			} else {
				slog.Info("PTY closed (EOF)")
			}
		}
		// PTY EOF - close WebSocket normally
		sendCloseMessage(conn, websocket.CloseNormalClosure, "PTY closed")
	}()

	// Goroutine 2: Read from WebSocket, write to PTY
	go func() {
		defer wg.Done()
		defer cancel()
		if err := webSocketToPTY(conn, ptyMaster, useBinaryMode); err != nil {
			slog.Error("WebSocket to PTY error", "error", err)
		}
		// WebSocket closed/error - kill PTY process
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				slog.Warn("failed to kill process", "error", err)
			}
		}
	}()

	// Goroutine 3: Send periodic pings and check idle timeout
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		idleTimer := time.NewTimer(idleTimeout)
		defer idleTimer.Stop()

		for {
			select {
			case <-ticker.C:
				if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
					slog.Warn("failed to set write deadline for ping", "error", err)
				}
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					slog.Warn("ping failed", "error", err)
					cancel()
					return
				}
			case <-idleTimer.C:
				slog.Info("idle timeout reached", "timeout", idleTimeout)
				sendCloseMessage(conn, websocket.CloseNormalClosure, "idle timeout")
				cancel()
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Wait()
	slog.Info("WebSocket connection closed", "remote", r.RemoteAddr)
}

// ptyToWebSocket reads from PTY and sends to WebSocket.
// In binary mode: sends raw binary frames.
// In JSON mode: sends {"type":"data","payload":"base64..."} messages.
func ptyToWebSocket(conn *websocket.Conn, ptyMaster *os.File, useBinaryMode bool) error {
	buf := make([]byte, ptyBufferSize)
	for {
		n, err := ptyMaster.Read(buf)
		if n > 0 {
			if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				slog.Warn("failed to set write deadline", "error", err)
			}
			if useBinaryMode {
				// Binary transparent mode: send raw bytes
				if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					return fmt.Errorf("failed to send binary message: %w", err)
				}
			} else {
				// JSON mode: not implemented for output in this version
				// For simplicity, always use binary for PTY output
				if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					return fmt.Errorf("failed to send binary message: %w", err)
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				return io.EOF
			}
			return fmt.Errorf("PTY read error: %w", err)
		}
	}
}

// webSocketToPTY reads from WebSocket and writes to PTY.
// In binary mode: expects raw binary frames or JSON resize messages.
// In JSON mode: expects {"type":"data","payload":"..."} or {"type":"resize",...}
func webSocketToPTY(conn *websocket.Conn, ptyMaster *os.File, useBinaryMode bool) error {
	conn.SetReadLimit(maxMessageSize)
	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				slog.Info("WebSocket closed normally")
				return nil
			}
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Warn("unexpected WebSocket close", "error", err)
			}
			return err
		}

		if useBinaryMode {
			// Binary mode: handle both binary frames and JSON resize messages
			switch messageType {
			case websocket.BinaryMessage:
				// Write raw binary data to PTY
				if _, err := ptyMaster.Write(data); err != nil {
					return fmt.Errorf("PTY write error: %w", err)
				}
			case websocket.TextMessage:
				// Check if it's a resize message
				var msg Message
				if err := json.Unmarshal(data, &msg); err == nil && msg.Type == "resize" {
					if msg.Cols > 0 && msg.Rows > 0 {
						if err := pty.SetWinsize(ptyMaster.Fd(), msg.Cols, msg.Rows); err != nil {
							slog.Warn("failed to resize PTY", "error", err)
						} else {
							slog.Debug("PTY resized", "cols", msg.Cols, "rows", msg.Rows)
						}
					}
				} else {
					// Treat as raw text and write to PTY
					if _, err := ptyMaster.Write(data); err != nil {
						return fmt.Errorf("PTY write error: %w", err)
					}
				}
			}
		} else {
			// JSON mode: expect JSON messages only
			var msg Message
			if err := json.Unmarshal(data, &msg); err != nil {
				slog.Warn("failed to parse JSON message", "error", err)
				continue
			}

			switch msg.Type {
			case "resize":
				if msg.Cols > 0 && msg.Rows > 0 {
					if err := pty.SetWinsize(ptyMaster.Fd(), msg.Cols, msg.Rows); err != nil {
						slog.Warn("failed to resize PTY", "error", err)
					} else {
						slog.Debug("PTY resized", "cols", msg.Cols, "rows", msg.Rows)
					}
				}
			default:
				slog.Warn("unknown message type in JSON mode", "type", msg.Type)
			}
		}
	}
}

// sendCloseMessage sends a close message to the WebSocket client.
func sendCloseMessage(conn *websocket.Conn, closeCode int, message string) {
	if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		slog.Warn("failed to set write deadline for close message", "error", err)
	}
	closeMsg := websocket.FormatCloseMessage(closeCode, message)
	if err := conn.WriteMessage(websocket.CloseMessage, closeMsg); err != nil {
		slog.Warn("failed to send close message", "error", err)
	}
}
