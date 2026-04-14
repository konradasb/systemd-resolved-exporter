package collector

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/konradasb/systemd-resolved-exporter/internal/resolver"
)

type stubProvider struct {
	stats *resolver.Stats
	err   error
}

func (s *stubProvider) GetStats(_ context.Context) (*resolver.Stats, error) {
	return s.stats, s.err
}

func TestCollectorEmitsAllMetrics(t *testing.T) {
	stats := &resolver.Stats{
		Transactions: resolver.Transactions{
			Current: 1, Total: 31, Timeouts: 2,
			TimeoutsServedStale: 3, FailedResponses: 4, FailedResponsesServedStale: 5,
		},
		Cache:  resolver.Cache{Size: 3, Hits: 7, Misses: 24},
		DNSSEC: resolver.DNSSEC{Secure: 9, Insecure: 8, Bogus: 6, Indeterminate: 2},
	}

	c := New(&stubProvider{stats: stats}, time.Second, nil)

	cases := []struct {
		metric string
		want   string
	}{
		{
			"systemd_resolved_exporter_up",
			`# HELP systemd_resolved_exporter_up Whether the last scrape of systemd-resolved succeeded (1) or failed (0).
# TYPE systemd_resolved_exporter_up gauge
systemd_resolved_exporter_up 1
`,
		},
		{
			"systemd_resolved_cache_hits_total",
			`# HELP systemd_resolved_cache_hits_total Total number of resolver cache hits.
# TYPE systemd_resolved_cache_hits_total counter
systemd_resolved_cache_hits_total 7
`,
		},
		{
			"systemd_resolved_dnssec_verdicts_total",
			`# HELP systemd_resolved_dnssec_verdicts_total Total number of DNSSEC verdicts, partitioned by result.
# TYPE systemd_resolved_dnssec_verdicts_total counter
systemd_resolved_dnssec_verdicts_total{result="bogus"} 6
systemd_resolved_dnssec_verdicts_total{result="indeterminate"} 2
systemd_resolved_dnssec_verdicts_total{result="insecure"} 8
systemd_resolved_dnssec_verdicts_total{result="secure"} 9
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.metric, func(t *testing.T) {
			if err := testutil.CollectAndCompare(c, strings.NewReader(tc.want), tc.metric); err != nil {
				t.Fatalf("metric %s: %v", tc.metric, err)
			}
		})
	}
}

func TestCollectorReportsScrapeFailureAsDown(t *testing.T) {
	c := New(&stubProvider{err: errors.New("boom")}, time.Second, nil)

	want := `# HELP systemd_resolved_exporter_up Whether the last scrape of systemd-resolved succeeded (1) or failed (0).
# TYPE systemd_resolved_exporter_up gauge
systemd_resolved_exporter_up 0
`
	if err := testutil.CollectAndCompare(c, strings.NewReader(want), "systemd_resolved_exporter_up"); err != nil {
		t.Fatalf("up metric: %v", err)
	}
}

func TestCollectorIncrementsScrapeErrorsOnFailure(t *testing.T) {
	c := New(&stubProvider{err: errors.New("boom")}, time.Second, nil)

	// Two preliminary failing scrapes; the CollectAndCompare below makes a
	// third, so we expect scrape_errors_total = 3.
	for range 2 {
		if n := testutil.CollectAndCount(c, "systemd_resolved_exporter_scrape_errors_total"); n != 1 {
			t.Fatalf("expected 1 series, got %d", n)
		}
	}

	want := `# HELP systemd_resolved_exporter_scrape_errors_total Total number of failed systemd-resolved scrapes.
# TYPE systemd_resolved_exporter_scrape_errors_total counter
systemd_resolved_exporter_scrape_errors_total 3
`
	if err := testutil.CollectAndCompare(c, strings.NewReader(want), "systemd_resolved_exporter_scrape_errors_total"); err != nil {
		t.Fatalf("scrape_errors_total: %v", err)
	}
}
