# Trivy with the vulnerability DB and the Java index DB pre-downloaded, so the
# trivy-image step does not download them on every promotion. The Java DB is required: Trivy aborts on any
# image containing Java artifacts when it is missing and updates are skipped.
# The DBs go stale: images.yaml rebuilds this image weekly.
# renovate: datasource=docker depName=aquasec/trivy
ARG TRIVY_VERSION=0.76.0

FROM aquasec/trivy:${TRIVY_VERSION}
LABEL org.opencontainers.image.source=https://github.com/34fathombelow/kargo-custom-steps \
      org.opencontainers.image.description="Trivy with a pre-baked vulnerability DB for the Kargo trivy-image custom step"
ENV TRIVY_CACHE_DIR=/opt/trivy-cache \
    TRIVY_SKIP_DB_UPDATE=true \
    TRIVY_SKIP_JAVA_DB_UPDATE=true \
    TRIVY_NO_PROGRESS=true \
    TRIVY_DISABLE_TELEMETRY=true \
    HOME=/tmp
RUN apk add --no-cache bash jq && \
    mkdir -p /opt/trivy-cache && \
    TRIVY_SKIP_DB_UPDATE=false trivy image --download-db-only && \
    TRIVY_SKIP_JAVA_DB_UPDATE=false trivy image --download-java-db-only && \
    chown -R 65532:65532 /opt/trivy-cache && chmod -R a+rwX /opt/trivy-cache
USER 65532:65532
