package resolver

import "testing"

func TestParseCLIStats(t *testing.T) {
	in := []byte(`{
		"transactions": {
			"currentTransactions": 1,
			"totalTransactions": 31,
			"totalTimeouts": 2,
			"totalTimeoutsServedStale": 3,
			"totalFailedResponses": 4,
			"totalFailedResponsesServedStale": 5
		},
		"cache":  {"size": 3, "hits": 7, "misses": 24},
		"dnssec": {"secure": 9, "insecure": 8, "bogus": 6, "indeterminate": 2}
	}`)

	got, err := parseCLIStats(in)
	if err != nil {
		t.Fatalf("parseCLIStats: %v", err)
	}

	want := &Stats{
		Transactions: Transactions{
			Current:                    1,
			Total:                      31,
			Timeouts:                   2,
			TimeoutsServedStale:        3,
			FailedResponses:            4,
			FailedResponsesServedStale: 5,
		},
		Cache:  Cache{Size: 3, Hits: 7, Misses: 24},
		DNSSEC: DNSSEC{Secure: 9, Insecure: 8, Bogus: 6, Indeterminate: 2},
	}
	if *got != *want {
		t.Fatalf("stats mismatch:\n got=%+v\nwant=%+v", got, want)
	}
}

func TestParseCLIStatsInvalid(t *testing.T) {
	if _, err := parseCLIStats([]byte("not-json")); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
