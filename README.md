# systemd-resolved-exporter

A Prometheus exporter for [`systemd-resolved`](https://www.freedesktop.org/software/systemd/man/systemd-resolved.service.html)
runtime statistics — DNS transactions, cache hits/misses, and DNSSEC verdicts.

Statistics are scraped on demand from systemd-resolved either over the system
D-Bus (default) or by invoking `resolvectl statistics --json=short`.

## Features

- Idiomatic Go (`log/slog`, Cobra, `prometheus/client_golang`).
- Pluggable collection backends: `dbus` (preferred) or `cli` fallback.
- Custom `prometheus.Collector` — no global state, safe for concurrent scrapes.
- TLS, mTLS, and basic-auth via [`prometheus/exporter-toolkit`](https://github.com/prometheus/exporter-toolkit).
- Standard exporter conventions: dotted flag names (`--web.listen-address`,
  `--log.level`, …), `systemd_resolved_exporter_build_info`,
  `systemd_resolved_exporter_up`, `..._scrape_duration_seconds`,
  `..._scrape_errors_total`.
- Per-scrape timeout and graceful shutdown on `SIGINT` / `SIGTERM`.
- Static binaries built by [GoReleaser](https://goreleaser.com/) for
  `linux/amd64`, `linux/arm64`, `linux/armv7`, plus multi-arch container
  images published to GHCR.

## Installation

### Container image

```sh
docker run --rm --net=host --user=nonroot \
  ghcr.io/konradasb/systemd-resolved-exporter:latest \
  --collect-mode=cli
```

### From a release archive

Download a tarball from the [releases page](https://github.com/konradasb/systemd-resolved-exporter/releases),
extract it, and put the binary on your `PATH`:

```sh
tar -xzf systemd-resolved-exporter_<version>_linux_amd64.tar.gz
sudo install -m 0755 systemd-resolved-exporter /usr/local/bin/
```

### From source

```sh
go install github.com/konradasb/systemd-resolved-exporter/cmd/systemd-resolved-exporter@latest
```

### Build locally

```sh
make build
./bin/systemd-resolved-exporter --help
```

## Usage

```sh
systemd-resolved-exporter \
  --web.listen-address=:9924 \
  --collect-mode=dbus \
  --log.level=info \
  --log.format=logfmt \
  --timeout=5s
```

### Flags

| Flag                    | Default     | Description                                                |
| ----------------------- | ----------- | ---------------------------------------------------------- |
| `--web.listen-address`  | `:9924`     | Address(es) to listen on. Repeatable.                      |
| `--web.telemetry-path`  | `/metrics`  | Path under which to expose metrics.                        |
| `--web.config.file`     | `""`        | Path to [exporter-toolkit web config][toolkit] for TLS/auth. |
| `--log.level`           | `info`      | `debug`, `info`, `warn`, `error`.                          |
| `--log.format`          | `logfmt`    | `logfmt` or `json`.                                        |
| `--collect-mode`        | `dbus`      | Collection backend (`dbus` or `cli`).                      |
| `--timeout`             | `5s`        | Per-scrape timeout.                                        |
| `--shutdown-timeout`    | `10s`       | Graceful shutdown deadline.                                |

[toolkit]: https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md

### Endpoints

| Path       | Description                  |
| ---------- | ---------------------------- |
| `/metrics` | Prometheus metrics           |
| `/healthz` | Liveness probe (always 200)  |
| `/`        | Landing page                 |

### TLS / authentication

Pass `--web.config.file=web-config.yml` with, for example:

```yaml
tls_server_config:
  cert_file: /etc/ssl/certs/exporter.crt
  key_file:  /etc/ssl/private/exporter.key

basic_auth_users:
  prometheus: $2y$12$<bcrypt-hash>
```

See the [exporter-toolkit web configuration docs][toolkit] for the full schema.

## Exposed metrics

### Target metrics (`systemd_resolved_*`)

| Metric                                                        | Type    | Description                                              |
| ------------------------------------------------------------- | ------- | -------------------------------------------------------- |
| `systemd_resolved_transactions_current`                       | gauge   | In-flight DNS transactions                               |
| `systemd_resolved_transactions_total`                         | counter | DNS transactions handled                                 |
| `systemd_resolved_transaction_timeouts_total`                 | counter | Transactions that timed out                              |
| `systemd_resolved_transaction_timeouts_served_stale_total`    | counter | Timeouts answered with stale data (CLI mode only)        |
| `systemd_resolved_failed_responses_total`                     | counter | Failed DNS responses (CLI mode only)                     |
| `systemd_resolved_failed_responses_served_stale_total`        | counter | Failed responses served stale (CLI mode only)            |
| `systemd_resolved_cache_entries`                              | gauge   | Current resolver cache entries                           |
| `systemd_resolved_cache_hits_total`                           | counter | Resolver cache hits                                      |
| `systemd_resolved_cache_misses_total`                         | counter | Resolver cache misses                                    |
| `systemd_resolved_dnssec_verdicts_total{result=...}`          | counter | DNSSEC verdicts: `secure`, `insecure`, `bogus`, `indeterminate` |

### Exporter-internal metrics (`systemd_resolved_exporter_*`)

| Metric                                            | Type    | Description                                |
| ------------------------------------------------- | ------- | ------------------------------------------ |
| `systemd_resolved_exporter_up`                    | gauge   | `1` if the last scrape succeeded           |
| `systemd_resolved_exporter_scrape_duration_seconds` | gauge | Duration of the last scrape                |
| `systemd_resolved_exporter_scrape_errors_total`   | counter | Total number of failed scrapes             |
| `systemd_resolved_exporter_build_info`            | gauge   | Build metadata exposed by `prometheus/common/version` |

> The newer "served stale" counters are only exposed by `resolvectl --json`,
> not by the D-Bus interface. In `dbus` mode they are reported as zero.

## Prometheus / Grafana

Example scrape config and alerting rules live under [`examples/`](examples/).
A starter Grafana dashboard is at [`dashboards/systemd-resolved.json`](dashboards/systemd-resolved.json).

```yaml
scrape_configs:
  - job_name: systemd-resolved
    static_configs:
      - targets: ["localhost:9924"]
```

## Running as a systemd service

A hardened unit file is shipped at
[`packaging/systemd-resolved-exporter.service`](packaging/systemd-resolved-exporter.service):

```sh
sudo install -m 0644 packaging/systemd-resolved-exporter.service \
  /etc/systemd/system/systemd-resolved-exporter.service
sudo systemctl daemon-reload
sudo systemctl enable --now systemd-resolved-exporter
```

## Default port

This exporter listens on **TCP/9924** by default. If/when this exporter is
adopted into `prometheus-community`, this port should be reserved in the
[Prometheus default port allocations](https://github.com/prometheus/prometheus/wiki/Default-port-allocations).

## Development

```sh
make all          # tidy, fmt, vet, lint, test, build
make snapshot     # local goreleaser snapshot (binaries + container images)
make run          # run with --collect-mode=cli at debug level
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the developer workflow and DCO
sign-off requirements.

## License

Licensed under the [Apache License, Version 2.0](LICENSE). See [NOTICE](NOTICE).
