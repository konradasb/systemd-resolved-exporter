# syntax=docker/dockerfile:1
#
# This Dockerfile is used by GoReleaser, which builds the binary first and
# then COPYs the prebuilt artifact in. For local builds use `make snapshot`.
FROM gcr.io/distroless/static-debian12:nonroot

ARG TARGETOS
ARG TARGETARCH

LABEL org.opencontainers.image.title="systemd-resolved-exporter" \
      org.opencontainers.image.description="Prometheus exporter for systemd-resolved statistics" \
      org.opencontainers.image.source="https://github.com/konradasb/systemd-resolved-exporter" \
      org.opencontainers.image.licenses="Apache-2.0"

COPY systemd-resolved-exporter /usr/bin/systemd-resolved-exporter

USER nonroot:nonroot
EXPOSE 9924
ENTRYPOINT ["/usr/bin/systemd-resolved-exporter"]
