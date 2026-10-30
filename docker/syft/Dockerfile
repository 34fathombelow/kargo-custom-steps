# Syft for the Kargo sbom-generate custom step: it pulls an image from its
# registry (no Docker daemon) and writes its SBOM. bash and jq summarize the SBOM
# and check it against the step's deny rules.
# renovate: datasource=docker depName=ghcr.io/anchore/syft
ARG SYFT_VERSION=1.55.0
# renovate: datasource=docker depName=alpine
ARG ALPINE_VERSION=3.24.2

FROM ghcr.io/anchore/syft:v${SYFT_VERSION} AS syft

FROM alpine:${ALPINE_VERSION}
ARG SYFT_VERSION
LABEL org.opencontainers.image.source=https://github.com/34fathombelow/kargo-custom-steps \
      org.opencontainers.image.description="Syft (SBOM generator) for the Kargo sbom-generate custom step" \
      org.opencontainers.image.version=${SYFT_VERSION}
RUN apk add --no-cache bash ca-certificates jq
COPY --from=syft /syft /usr/local/bin/syft
# Custom steps run as 65532 with a read-only view of most of the filesystem;
# Syft unpacks image layers and caches under a writable /tmp.
ENV HOME=/tmp \
    TMPDIR=/tmp \
    SYFT_CHECK_FOR_APP_UPDATE=false
USER 65532:65532
