// Package resolver defines the data model and collection contract for
// systemd-resolved statistics.
package resolver

import "context"

// Stats is a snapshot of systemd-resolved runtime statistics.
type Stats struct {
	Transactions Transactions
	Cache        Cache
	DNSSEC       DNSSEC
}

// Transactions captures DNS transaction counters.
type Transactions struct {
	Current                    uint64
	Total                      uint64
	Timeouts                   uint64
	TimeoutsServedStale        uint64
	FailedResponses            uint64
	FailedResponsesServedStale uint64
}

// Cache captures resolver cache statistics.
type Cache struct {
	Size   uint64
	Hits   uint64
	Misses uint64
}

// DNSSEC captures DNSSEC verdict counters.
type DNSSEC struct {
	Secure        uint64
	Insecure      uint64
	Bogus         uint64
	Indeterminate uint64
}

// Provider abstracts the source of systemd-resolved statistics so the
// collector can be exercised against multiple backends and tests.
type Provider interface {
	GetStats(ctx context.Context) (*Stats, error)
}
