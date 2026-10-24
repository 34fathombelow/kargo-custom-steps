# Kyverno CLI plus a shell and jq, so kyverno-apply can summarize the policy report.
# renovate: datasource=docker depName=ghcr.io/kyverno/kyverno-cli
ARG KYVERNO_VERSION=v1.19.1
# renovate: datasource=docker depName=alpine
ARG ALPINE_VERSION=3.24.2

FROM ghcr.io/kyverno/kyverno-cli:${KYVERNO_VERSION} AS kyverno

FROM alpine:${ALPINE_VERSION}
LABEL org.opencontainers.image.source=https://github.com/34fathombelow/kargo-custom-steps \
      org.opencontainers.image.description="Kyverno CLI for the Kargo kyverno-apply custom step"
RUN apk add --no-cache bash ca-certificates jq
COPY --from=kyverno /ko-app/kubectl-kyverno /usr/local/bin/kyverno
ENV HOME=/tmp
USER 65532:65532
