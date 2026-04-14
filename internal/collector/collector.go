// Package collector implements a Prometheus collector that publishes
// systemd-resolved statistics fetched from a resolver.Provider.
//
// Two metric namespaces are used:
//
//   - "systemd_resolved_*"          — properties of the resolver itself.
//   - "systemd_resolved_exporter_*" — properties of this exporter's scrape.
//
// This split follows the Prometheus exporter writing guide, where target
// metrics and exporter-internal metrics are kept under distinct namespaces.
package collector

import (
	"context"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/konradasb/systemd-resolved-exporter/internal/resolver"
)

const (
	namespace         = "systemd_resolved"
	exporterNamespace = "systemd_resolved_exporter"
)

// Collector turns a resolver.Provider into Prometheus metrics. It is safe
// for concurrent scrapes because the only mutable state is the cumulative
// scrape-error counter, which is itself goroutine-safe.
type Collector struct {
	provider resolver.Provider
	timeout  time.Duration
	logger   *slog.Logger

	// Exporter-internal metrics.
	up             *prometheus.Desc
	scrapeDuration *prometheus.Desc
	scrapeErrors   prometheus.Counter

	// Target metrics.
	txCurrent           *prometheus.Desc
	txTotal             *prometheus.Desc
	txTimeouts          *prometheus.Desc
	txTimeoutsStale     *prometheus.Desc
	failedResponses     *prometheus.Desc
	failedResponsesStal *prometheus.Desc
	cacheSize           *prometheus.Desc
	cacheHits           *prometheus.Desc
	cacheMisses         *prometheus.Desc
	dnssecVerdicts      *prometheus.Desc
}

// New constructs a Collector. A nil logger falls back to slog.Default.
func New(p resolver.Provider, timeout time.Duration, logger *slog.Logger) *Collector {
	if logger == nil {
		logger = slog.Default()
	}
	desc := func(ns, name, help string, labels ...string) *prometheus.Desc {
		return prometheus.NewDesc(ns+"_"+name, help, labels, nil)
	}
	return &Collector{
		provider: p,
		timeout:  timeout,
		logger:   logger,

		up: desc(exporterNamespace, "up",
			"Whether the last scrape of systemd-resolved succeeded (1) or failed (0)."),
		scrapeDuration: desc(exporterNamespace, "scrape_duration_seconds",
			"Duration of the last systemd-resolved scrape in seconds."),
		scrapeErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: exporterNamespace,
			Name:      "scrape_errors_total",
			Help:      "Total number of failed systemd-resolved scrapes.",
		}),

		txCurrent: desc(namespace, "transactions_current",
			"Number of currently in-flight DNS transactions."),
		txTotal: desc(namespace, "transactions_total",
			"Total number of DNS transactions handled since startup."),
		txTimeouts: desc(namespace, "transaction_timeouts_total",
			"Total number of DNS transactions that timed out."),
		txTimeoutsStale: desc(namespace, "transaction_timeouts_served_stale_total",
			"Total number of timed-out transactions answered with stale data."),
		failedResponses: desc(namespace, "failed_responses_total",
			"Total number of failed DNS responses."),
		failedResponsesStal: desc(namespace, "failed_responses_served_stale_total",
			"Total number of failed responses answered with stale data."),
		cacheSize: desc(namespace, "cache_entries",
			"Number of entries currently held in the resolver cache."),
		cacheHits: desc(namespace, "cache_hits_total",
			"Total number of resolver cache hits."),
		cacheMisses: desc(namespace, "cache_misses_total",
			"Total number of resolver cache misses."),
		dnssecVerdicts: desc(namespace, "dnssec_verdicts_total",
			"Total number of DNSSEC verdicts, partitioned by result.", "result"),
	}
}

// Describe implements prometheus.Collector.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range []*prometheus.Desc{
		c.up, c.scrapeDuration,
		c.txCurrent, c.txTotal, c.txTimeouts, c.txTimeoutsStale,
		c.failedResponses, c.failedResponsesStal,
		c.cacheSize, c.cacheHits, c.cacheMisses,
		c.dnssecVerdicts,
	} {
		ch <- d
	}
	c.scrapeErrors.Describe(ch)
}

// Collect implements prometheus.Collector. Scrape failures surface as
// systemd_resolved_exporter_up=0 and increment scrape_errors_total; the
// process never returns an error to the Prometheus handler since that would
// drop the partial scrape.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	start := time.Now()
	stats, err := c.provider.GetStats(ctx)
	elapsed := time.Since(start).Seconds()

	ch <- prometheus.MustNewConstMetric(c.scrapeDuration, prometheus.GaugeValue, elapsed)
	if err != nil {
		c.logger.ErrorContext(ctx, "scrape failed", "error", err)
		c.scrapeErrors.Inc()
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
		c.scrapeErrors.Collect(ch)
		return
	}
	ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 1)
	c.scrapeErrors.Collect(ch)

	emit := func(d *prometheus.Desc, t prometheus.ValueType, v uint64, labels ...string) {
		ch <- prometheus.MustNewConstMetric(d, t, float64(v), labels...)
	}

	emit(c.txCurrent, prometheus.GaugeValue, stats.Transactions.Current)
	emit(c.txTotal, prometheus.CounterValue, stats.Transactions.Total)
	emit(c.txTimeouts, prometheus.CounterValue, stats.Transactions.Timeouts)
	emit(c.txTimeoutsStale, prometheus.CounterValue, stats.Transactions.TimeoutsServedStale)
	emit(c.failedResponses, prometheus.CounterValue, stats.Transactions.FailedResponses)
	emit(c.failedResponsesStal, prometheus.CounterValue, stats.Transactions.FailedResponsesServedStale)

	emit(c.cacheSize, prometheus.GaugeValue, stats.Cache.Size)
	emit(c.cacheHits, prometheus.CounterValue, stats.Cache.Hits)
	emit(c.cacheMisses, prometheus.CounterValue, stats.Cache.Misses)

	emit(c.dnssecVerdicts, prometheus.CounterValue, stats.DNSSEC.Secure, "secure")
	emit(c.dnssecVerdicts, prometheus.CounterValue, stats.DNSSEC.Insecure, "insecure")
	emit(c.dnssecVerdicts, prometheus.CounterValue, stats.DNSSEC.Bogus, "bogus")
	emit(c.dnssecVerdicts, prometheus.CounterValue, stats.DNSSEC.Indeterminate, "indeterminate")
}
