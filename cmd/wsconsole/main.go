//go:build linux
// +build linux

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danmaid/wsconsole/internal/ws"
)

var (
	addr             = flag.String("addr", ":8080", "HTTP service address")
	staticDir        = flag.String("static", "./deploy/static", "Static files directory")
	logLevel         = flag.String("log", "info", "Log level (debug, info, warn, error)")
	launcherStrategy = flag.String("launcher", "auto", "Login launcher strategy: auto (default), direct (UID=0), or systemd-run")
)

// loggingMiddleware wraps an HTTP handler with request/response logging
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"remote", r.RemoteAddr,
			"duration_ms", duration.Milliseconds(),
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
// It also implements http.Hijacker for WebSocket support
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Hijack implements http.Hijacker interface for WebSocket upgrade
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("ResponseWriter does not implement http.Hijacker")
	}
	return hijacker.Hijack()
}

func main() {
	flag.Parse()

	// Setup structured logging
	level := slog.LevelInfo
	switch *logLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	slog.Info("starting wsconsole server", "addr", *addr, "launcher_strategy", *launcherStrategy)

	// Setup HTTP routes
	mux := http.NewServeMux()

	// WebSocket endpoint with launcher strategy parameter
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// Override strategy from query parameter if provided, otherwise use CLI flag
		query := r.URL.Query()
		if query.Get("launcher") == "" && *launcherStrategy != "auto" {
			// Add launcher strategy to query if not already present
			query.Set("launcher", *launcherStrategy)
			r.URL.RawQuery = query.Encode()
		}
		ws.Handler(w, r)
	})

	// Health check endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {

			slog.Warn("failed to write healthz response", "error", err)

		}
	})

	// Serve static files
	if info, err := os.Stat(*staticDir); err == nil && info.IsDir() {
		slog.Info("serving static files", "dir", *staticDir)
		fs := http.FileServer(http.Dir(*staticDir))
		mux.Handle("/", fs)
	} else {
		slog.Warn("static directory not found, static files disabled", "dir", *staticDir)
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				html := `<!DOCTYPE html>
<html>
<head><title>wsconsole</title></head>
<body>
<h1>wsconsole is running</h1>
<p>Connect to <a href="/ws">/ws</a> via WebSocket</p>
<p>Health check: <a href="/healthz">/healthz</a></p>
</body>
</html>`
				if _, err := w.Write([]byte(html)); err != nil {
					slog.Warn("failed to write index.html response", "error", err)
				}
			} else {
				http.NotFound(w, r)
			}
		})
	}

	// Add HTTP access logging middleware
	handler := loggingMiddleware(mux)

	// Create HTTP server
	server := &http.Server{
		Addr:         *addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		slog.Info("HTTP server listening", "addr", *addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}
