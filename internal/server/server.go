// Package server wires the Prometheus registry into an HTTP server backed
// by prometheus/exporter-toolkit, providing TLS, basic-auth, the standard
// landing page, and graceful shutdown.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
)

// Config controls the HTTP listener and exporter-toolkit web config.
type Config struct {
	ListenAddresses []string
	TelemetryPath   string
	WebConfigFile   string
	ShutdownTimeout time.Duration
}

// Server wraps an *http.Server with exporter-toolkit's listener and
// graceful-shutdown semantics.
type Server struct {
	cfg    Config
	http   *http.Server
	logger *slog.Logger
}

// New constructs a Server with the given registry exposed at TelemetryPath.
func New(cfg Config, registry *prometheus.Registry, logger *slog.Logger) (*Server, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.TelemetryPath == "" {
		cfg.TelemetryPath = "/metrics"
	}
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = 10 * time.Second
	}

	mux := http.NewServeMux()
	mux.Handle(cfg.TelemetryPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		Registry:          registry,
		EnableOpenMetrics: true,
	}))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	if cfg.TelemetryPath != "/" {
		landing, err := web.NewLandingPage(web.LandingConfig{
			Name:        "systemd-resolved Exporter",
			Description: "Prometheus exporter for systemd-resolved statistics.",
			Version:     version.Info(),
			Links: []web.LandingLinks{
				{Address: cfg.TelemetryPath, Text: "Metrics"},
				{Address: "/healthz", Text: "Health"},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("build landing page: %w", err)
		}
		mux.Handle("/", landing)
	}

	return &Server{
		cfg:    cfg,
		logger: logger,
		http: &http.Server{
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}, nil
}

// Run blocks until ctx is cancelled or the listener fails. On cancellation
// it performs a bounded graceful shutdown.
func (s *Server) Run(ctx context.Context) error {
	systemdSocket := false
	flagConfig := &web.FlagConfig{
		WebListenAddresses: &s.cfg.ListenAddresses,
		WebSystemdSocket:   &systemdSocket,
		WebConfigFile:      &s.cfg.WebConfigFile,
	}

	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("http listener started", "addresses", s.cfg.ListenAddresses)
		err := web.ListenAndServe(s.http, flagConfig, s.logger)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		errCh <- err
	}()

	select {
	case <-ctx.Done():
		s.logger.Info("shutdown requested")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
		defer cancel()
		if err := s.http.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		return <-errCh
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("http listener: %w", err)
		}
		return nil
	}
}
