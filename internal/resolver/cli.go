package resolver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// CLIProvider collects statistics by shelling out to `resolvectl`.
//
// It is the most portable backend: it works anywhere resolvectl is on PATH
// and does not require system bus access from the exporter process.
type CLIProvider struct {
	binary string
}

// NewCLIProvider returns a CLIProvider that invokes the given binary.
// An empty binary defaults to "resolvectl" looked up on PATH.
func NewCLIProvider(binary string) *CLIProvider {
	if binary == "" {
		binary = "resolvectl"
	}
	return &CLIProvider{binary: binary}
}

type cliPayload struct {
	Transactions struct {
		Current                    uint64 `json:"currentTransactions"`
		Total                      uint64 `json:"totalTransactions"`
		Timeouts                   uint64 `json:"totalTimeouts"`
		TimeoutsServedStale        uint64 `json:"totalTimeoutsServedStale"`
		FailedResponses            uint64 `json:"totalFailedResponses"`
		FailedResponsesServedStale uint64 `json:"totalFailedResponsesServedStale"`
	} `json:"transactions"`
	Cache struct {
		Size   uint64 `json:"size"`
		Hits   uint64 `json:"hits"`
		Misses uint64 `json:"misses"`
	} `json:"cache"`
	DNSSEC struct {
		Secure        uint64 `json:"secure"`
		Insecure      uint64 `json:"insecure"`
		Bogus         uint64 `json:"bogus"`
		Indeterminate uint64 `json:"indeterminate"`
	} `json:"dnssec"`
}

// GetStats runs `resolvectl statistics --json=short` under the given context
// and decodes the resulting JSON document.
func (p *CLIProvider) GetStats(ctx context.Context) (*Stats, error) {
	cmd := exec.CommandContext(ctx, p.binary, "statistics", "--json=short")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run %s: %w (stderr=%q)", p.binary, err, stderr.String())
	}
	return parseCLIStats(stdout.Bytes())
}

func parseCLIStats(data []byte) (*Stats, error) {
	var p cliPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("decode resolvectl output: %w", err)
	}
	return &Stats{
		Transactions: Transactions{
			Current:                    p.Transactions.Current,
			Total:                      p.Transactions.Total,
			Timeouts:                   p.Transactions.Timeouts,
			TimeoutsServedStale:        p.Transactions.TimeoutsServedStale,
			FailedResponses:            p.Transactions.FailedResponses,
			FailedResponsesServedStale: p.Transactions.FailedResponsesServedStale,
		},
		Cache: Cache{
			Size:   p.Cache.Size,
			Hits:   p.Cache.Hits,
			Misses: p.Cache.Misses,
		},
		DNSSEC: DNSSEC{
			Secure:        p.DNSSEC.Secure,
			Insecure:      p.DNSSEC.Insecure,
			Bogus:         p.DNSSEC.Bogus,
			Indeterminate: p.DNSSEC.Indeterminate,
		},
	}, nil
}
