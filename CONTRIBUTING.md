# Contributing

Thanks for your interest in improving systemd-resolved-exporter!

## Ground rules

- Be kind. This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md).
- Discuss large changes in an issue first so we can agree on scope.
- Keep PRs small and focused; one logical change per PR.

## Developer Certificate of Origin

All commits must be signed off under the [Developer Certificate of Origin](https://developercertificate.org/).
Use `git commit -s` (or `git commit --signoff`) to add a `Signed-off-by:` trailer.
A CI check enforces this on every pull request.

## Development workflow

```sh
git clone https://github.com/konradasb/systemd-resolved-exporter
cd systemd-resolved-exporter
make all          # tidy, fmt, vet, lint, test, build
```

Useful targets:

| Target           | Description                                |
| ---------------- | ------------------------------------------ |
| `make test`      | Run unit tests with race detector          |
| `make lint`      | Run `golangci-lint`                        |
| `make build`     | Build the binary into `./bin`              |
| `make snapshot`  | Local goreleaser snapshot build            |
| `make run`       | Run with `--collect-mode=cli` and debug    |

## Coding style

- Standard `gofmt` / `goimports` (run `make fmt`).
- Errors are wrapped with `fmt.Errorf("...: %w", err)`.
- Comments follow GoDoc conventions on every exported symbol.
- Commits use [Conventional Commits](https://www.conventionalcommits.org/),
  e.g. `feat(collector): expose stale-served counters`.

## Reporting bugs

Open a [bug report](https://github.com/konradasb/systemd-resolved-exporter/issues/new/choose)
and fill in the template. Include `systemd --version`, the exporter version,
and the output of `resolvectl statistics --json=pretty` if you can.

## Security issues

Please do not file public issues for security vulnerabilities.
See [SECURITY.md](SECURITY.md) for the disclosure process.
