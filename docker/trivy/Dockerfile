# Trivy plus bash and jq for the trivy-image step, running as 65532.
# No vulnerability DB is baked in: each scan downloads the current DB (about
# 130 MiB) and, only for images containing Java artifacts, the Java DB (about
# 940 MiB), so a security gate never scans with a stale DB. Both come from
# Trivy's default mirror (mirror.gcr.io) into a cache under /tmp.
# renovate: datasource=docker depName=aquasec/trivy
ARG TRIVY_VERSION=0.76.0

FROM aquasec/trivy:${TRIVY_VERSION}
LABEL org.opencontainers.image.source=https://github.com/34fathombelow/kargo-custom-steps \
      org.opencontainers.image.description="Trivy for the Kargo trivy-image custom step"
RUN apk add --no-cache bash jq
ENV TRIVY_CACHE_DIR=/tmp/trivy-cache \
    TRIVY_NO_PROGRESS=true \
    TRIVY_DISABLE_TELEMETRY=true \
    HOME=/tmp
USER 65532:65532
