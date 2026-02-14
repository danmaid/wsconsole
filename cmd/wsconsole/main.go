//go:build linux
// +build linux

package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/danmaid/wsconsole/internal/ws"
)

var (
	addr             = flag.String("addr", ":6001", "Server address (default :6001 for HTTPS)")
	staticDir        = flag.String("static", "./deploy/static", "Static files directory")
	logLevel         = flag.String("log", "info", "Log level (debug, info, warn, error)")
	launcherStrategy = flag.String("launcher", "auto", "Login launcher strategy: auto (default), direct (UID=0), or systemd-run")
	version          = flag.Bool("version", false, "Show version information")
	tlsEnabled       = flag.Bool("tls", true, "Enable TLS/HTTPS (default: true)")
	certFile         = flag.String("cert", "", "Path to TLS certificate file (auto-generated if empty and TLS enabled)")
	keyFile          = flag.String("key", "", "Path to TLS key file (auto-generated if empty and TLS enabled)")
	pathPrefix       = flag.String("path-prefix", "", "Path prefix for reverse proxy setup (e.g., /wsconsole)")
)

// Version is set during build with -ldflags
var Version = "0.0.1"

// generateSelfSignedCert generates a self-signed certificate and returns cert/key file paths
func generateSelfSignedCert() (string, string, error) {
	// Create certificates directory
	certDir := filepath.Join(os.TempDir(), "wsconsole")
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return "", "", fmt.Errorf("failed to create cert directory: %w", err)
	}

	certPath := filepath.Join(certDir, "cert.pem")
	keyPath := filepath.Join(certDir, "key.pem")

	// Check if cert already exists
	if _, err := os.Stat(certPath); err == nil {
		if _, err := os.Stat(keyPath); err == nil {
			slog.Info("using existing self-signed certificate", "cert", certPath, "key", keyPath)
			return certPath, keyPath, nil
		}
	}

	slog.Info("generating self-signed certificate", "cert", certPath, "key", keyPath)

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "localhost",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(1, 0, 0), // Valid for 1 year
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		DNSNames: []string{"localhost", "127.0.0.1"},
		IPAddresses: []net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP("::1"),
		},
	}

	// Create self-signed certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to create certificate: %w", err)
	}

	// Write certificate to file
	certOut, err := os.Create(certPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create cert file: %w", err)
	}
	defer func() {
		if err := certOut.Close(); err != nil {
			slog.Warn("failed to close cert file", "error", err)
		}
	}()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return "", "", fmt.Errorf("failed to encode certificate: %w", err)
	}

	// Write private key to file
	keyOut, err := os.Create(keyPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create key file: %w", err)
	}
	defer func() {
		if err := keyOut.Close(); err != nil {
			slog.Warn("failed to close key file", "error", err)
		}
	}()

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateKeyBytes}); err != nil {
		return "", "", fmt.Errorf("failed to encode private key: %w", err)
	}

	return certPath, keyPath, nil
}

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

	// Handle version flag
	if *version {
		fmt.Printf("wsconsole version %s\n", Version)
		os.Exit(0)
	}

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

	slog.Info("starting wsconsole server",
		"version", Version,
		"addr", *addr,
		"tls_enabled", *tlsEnabled,
		"path_prefix", *pathPrefix,
		"launcher_strategy", *launcherStrategy)

	// Normalize path prefix
	prefix := strings.TrimSuffix(strings.TrimSpace(*pathPrefix), "/")
	if prefix != "" && !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	// Setup HTTP routes
	mux := http.NewServeMux()

	// WebSocket endpoint with launcher strategy parameter
	wsPath := prefix + "/ws"
	mux.HandleFunc(wsPath, func(w http.ResponseWriter, r *http.Request) {
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
	healthPath := prefix + "/healthz"
	mux.HandleFunc(healthPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {

			slog.Warn("failed to write healthz response", "error", err)

		}
	})

	// Serve static files
	if info, err := os.Stat(*staticDir); err == nil && info.IsDir() {
		slog.Info("serving static files", "dir", *staticDir)
		fs := http.FileServer(http.Dir(*staticDir))
		indexPath := prefix + "/"
		mux.Handle(indexPath, fs)
	} else {
		slog.Warn("static directory not found, static files disabled", "dir", *staticDir)
		mux.HandleFunc(prefix+"/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == prefix+"/" || r.URL.Path == prefix {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				protocol := "wss"
				if !*tlsEnabled {
					protocol = "ws"
				}
				wsEndpoint := fmt.Sprintf("%s://%s%s/ws", protocol, r.Host, prefix)
				html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>wsconsole</title></head>
<body>
<h1>wsconsole is running</h1>
<p>WebSocket endpoint: <a href="%s">%s</a></p>
<p>Health check: <a href="%s/healthz">%s/healthz</a></p>
</body>
</html>`, wsEndpoint, wsEndpoint, prefix, prefix)
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
		if *tlsEnabled {
			// Determine certificate files
			cert := *certFile
			key := *keyFile

			if cert == "" || key == "" {
				// Auto-generate self-signed certificate
				var err error
				cert, key, err = generateSelfSignedCert()
				if err != nil {
					slog.Error("failed to generate self-signed certificate", "error", err)
					os.Exit(1)
				}
			}

			// Verify certificate files exist
			if _, err := os.Stat(cert); err != nil {
				slog.Error("certificate file not found", "path", cert, "error", err)
				os.Exit(1)
			}
			if _, err := os.Stat(key); err != nil {
				slog.Error("key file not found", "path", key, "error", err)
				os.Exit(1)
			}

			slog.Info("HTTPS server listening", "addr", *addr, "cert", cert, "key", key)
			if err := server.ListenAndServeTLS(cert, key); err != nil && err != http.ErrServerClosed {
				slog.Error("server error", "error", err)
				os.Exit(1)
			}
		} else {
			slog.Info("HTTP server listening", "addr", *addr)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("server error", "error", err)
				os.Exit(1)
			}
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
