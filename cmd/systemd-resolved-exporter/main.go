// Command systemd-resolved-exporter exposes systemd-resolved runtime
// statistics as Prometheus metrics over HTTP.
//
// Build-time variables exposed via -ldflags on
// github.com/prometheus/common/version are surfaced via the
// systemd_resolved_exporter_build_info metric and the --version flag.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/konradasb/systemd-resolved-exporter/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
