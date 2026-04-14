# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Initial implementation of the systemd-resolved Prometheus exporter.
- Pluggable collection backends (`dbus` and `cli`).
- TLS, mTLS and basic-auth via `prometheus/exporter-toolkit`.
- `systemd_resolved_exporter_build_info` metric via `prometheus/common/version`.
- Apache-2.0 license, NOTICE, CHANGELOG, and contributor docs.
- GoReleaser-based multi-arch release workflow with container images.

[Unreleased]: https://github.com/konradasb/systemd-resolved-exporter/commits/main
