package resolver

import (
	"context"
	"fmt"

	"github.com/godbus/dbus/v5"
)

const (
	dbusService = "org.freedesktop.resolve1"
	dbusPath    = "/org/freedesktop/resolve1"
	dbusIface   = "org.freedesktop.resolve1.Manager"
)

// DBusProvider collects statistics directly from systemd-resolved over the
// system D-Bus, avoiding the cost of forking a child process per scrape.
//
// Note: the D-Bus interface only exposes the historical counters; the newer
// "served stale" fields surfaced by `resolvectl --json` are not available
// here and will be reported as zero.
type DBusProvider struct {
	conn *dbus.Conn
}

// NewDBusProvider connects to the system bus.
func NewDBusProvider() (*DBusProvider, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, fmt.Errorf("connect system bus: %w", err)
	}
	return &DBusProvider{conn: conn}, nil
}

// Close releases the D-Bus connection.
func (p *DBusProvider) Close() error {
	if p.conn == nil {
		return nil
	}
	return p.conn.Close()
}

// GetStats reads the TransactionStatistics, CacheStatistics and
// DNSSECStatistics properties from the resolve1 manager.
func (p *DBusProvider) GetStats(ctx context.Context) (*Stats, error) {
	type result struct {
		stats *Stats
		err   error
	}
	done := make(chan result, 1)
	go func() {
		s, err := p.fetch()
		done <- result{stats: s, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-done:
		return r.stats, r.err
	}
}

func (p *DBusProvider) fetch() (*Stats, error) {
	obj := p.conn.Object(dbusService, dbusPath)

	tx, err := readUints(obj, dbusIface+".TransactionStatistics", 2)
	if err != nil {
		return nil, fmt.Errorf("read TransactionStatistics: %w", err)
	}
	cache, err := readUints(obj, dbusIface+".CacheStatistics", 3)
	if err != nil {
		return nil, fmt.Errorf("read CacheStatistics: %w", err)
	}
	sec, err := readUints(obj, dbusIface+".DNSSECStatistics", 4)
	if err != nil {
		return nil, fmt.Errorf("read DNSSECStatistics: %w", err)
	}

	return &Stats{
		Transactions: Transactions{Current: tx[0], Total: tx[1]},
		Cache:        Cache{Size: cache[0], Hits: cache[1], Misses: cache[2]},
		DNSSEC: DNSSEC{
			Secure:        sec[0],
			Insecure:      sec[1],
			Bogus:         sec[2],
			Indeterminate: sec[3],
		},
	}, nil
}

func readUints(obj dbus.BusObject, property string, n int) ([]uint64, error) {
	v, err := obj.GetProperty(property)
	if err != nil {
		return nil, err
	}
	raw, ok := v.Value().([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected type %T for %s", v.Value(), property)
	}
	if len(raw) != n {
		return nil, fmt.Errorf("expected %d fields in %s, got %d", n, property, len(raw))
	}
	out := make([]uint64, n)
	for i, x := range raw {
		u, ok := x.(uint64)
		if !ok {
			return nil, fmt.Errorf("field %d of %s is %T, want uint64", i, property, x)
		}
		out[i] = u
	}
	return out, nil
}
