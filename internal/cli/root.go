// Package cli builds the Cobra command tree for systemd-resolved-exporter
// and wires the collector, registry, and HTTP server together.
package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/version"
	"github.com/spf13/cobra"

	"github.com/konradasb/systemd-resolved-exporter/internal/collector"
	"github.com/konradasb/systemd-resolved-exporter/internal/resolver"
	"github.com/konradasb/systemd-resolved-exporter/internal/server"
)

const exporterName = "systemd_resolved_exporter"

type options struct {
	listenAddresses []string
	telemetryPath   string
	webConfigFile   string

	logLevel  string
	logFormat string

	collectMode     string
	timeout         time.Duration
	shutdownTimeout time.Duration
}

func (o *options) validate() error {
	if len(o.listenAddresses) == 0 {
		return fmt.Errorf("--web.listen-address must not be empty")
	}
	if o.telemetryPath == "" {
		return fmt.Errorf("--web.telemetry-path must not be empty")
	}
	if o.timeout <= 0 {
		return fmt.Errorf("--timeout must be positive")
	}
	switch strings.ToLower(o.collectMode) {
	case "dbus", "cli":
	default:
		return fmt.Errorf("--collect-mode must be one of: dbus, cli")
	}
	return nil
}

// NewRootCmd returns the root cobra command. Build-time version metadata
// (Version, Revision, Branch, BuildUser, BuildDate) is read from
// github.com/prometheus/common/version.
func NewRootCmd() *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:           "systemd-resolved-exporter",
		Short:         "Prometheus exporter for systemd-resolved statistics",
		Long:          "Exports DNS transaction, cache, and DNSSEC statistics from systemd-resolved as Prometheus metrics.",
		Version:       version.Print(exporterName),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := opts.validate(); err != nil {
				return err
			}
			return run(cmd.Context(), opts)
		},
	}
	cmd.SetVersionTemplate("{{.Version}}\n")

	f := cmd.Flags()
	f.StringSliceVar(&opts.listenAddresses, "web.listen-address",
		[]string{":9924"}, "Addresses on which to expose metrics and web interface. Repeatable for multiple listeners.")
	f.StringVar(&opts.telemetryPath, "web.telemetry-path",
		"/metrics", "Path under which to expose metrics.")
	f.StringVar(&opts.webConfigFile, "web.config.file",
		"", "Path to configuration file that can enable TLS or authentication. See https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md")

	f.StringVar(&opts.logLevel, "log.level", "info",
		"Only log messages with the given severity or above. One of: [debug, info, warn, error]")
	f.StringVar(&opts.logFormat, "log.format", "logfmt",
		"Output format of log messages. One of: [logfmt, json]")

	f.StringVar(&opts.collectMode, "collect-mode", "dbus",
		"Collection backend: dbus or cli")
	f.DurationVar(&opts.timeout, "timeout", 5*time.Second,
		"Timeout for a single scrape")
	f.DurationVar(&opts.shutdownTimeout, "shutdown-timeout", 10*time.Second,
		"Graceful shutdown timeout")

	return cmd
}

func run(ctx context.Context, opts *options) error {
	logger, err := newLogger(opts.logLevel, opts.logFormat)
	if err != nil {
		return err
	}
	slog.SetDefault(logger)

	provider, closer, err := buildProvider(opts.collectMode)
	if err != nil {
		return fmt.Errorf("init provider: %w", err)
	}
	if closer != nil {
		defer closer()
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		versioncollector.NewCollector(exporterName),
		collector.New(provider, opts.timeout, logger),
	)

	srv, err := server.New(server.Config{
		ListenAddresses: opts.listenAddresses,
		TelemetryPath:   opts.telemetryPath,
		WebConfigFile:   opts.webConfigFile,
		ShutdownTimeout: opts.shutdownTimeout,
	}, registry, logger)
	if err != nil {
		return fmt.Errorf("init http server: %w", err)
	}

	signalCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("starting systemd-resolved exporter",
		"version", version.Version,
		"revision", version.Revision,
		"listen", opts.listenAddresses,
		"mode", opts.collectMode,
	)

	return srv.Run(signalCtx)
}

func buildProvider(mode string) (resolver.Provider, func(), error) {
	switch strings.ToLower(mode) {
	case "dbus":
		p, err := resolver.NewDBusProvider()
		if err != nil {
			return nil, nil, err
		}
		return p, func() { _ = p.Close() }, nil
	case "cli":
		return resolver.NewCLIProvider(""), nil, nil
	default:
		return nil, nil, fmt.Errorf("unknown collect-mode %q", mode)
	}
}

func newLogger(level, format string) (*slog.Logger, error) {
	cfg := &promslog.Config{
		Level:  promslog.NewLevel(),
		Format: promslog.NewFormat(),
	}
	if err := cfg.Level.Set(strings.ToLower(level)); err != nil {
		return nil, fmt.Errorf("invalid --log.level: %w", err)
	}
	if err := cfg.Format.Set(strings.ToLower(format)); err != nil {
		return nil, fmt.Errorf("invalid --log.format: %w", err)
	}
	return promslog.New(cfg), nil
}
