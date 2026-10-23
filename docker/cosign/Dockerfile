# cosign plus a shell and jq, so the cosign-verify step can shape its output.
# The upstream image is distroless (no shell).
# renovate: datasource=docker depName=ghcr.io/sigstore/cosign/cosign
ARG COSIGN_VERSION=v3.1.3
# renovate: datasource=docker depName=alpine
ARG ALPINE_VERSION=3.24.2

FROM ghcr.io/sigstore/cosign/cosign:${COSIGN_VERSION} AS cosign

FROM alpine:${ALPINE_VERSION}
LABEL org.opencontainers.image.source=https://github.com/34fathombelow/kargo-custom-steps \
      org.opencontainers.image.description="cosign for the Kargo cosign-verify custom step"
# openssl reads signer identities (SAN, OIDC issuer) from the signature certificates.
RUN apk add --no-cache bash ca-certificates jq openssl
COPY --from=cosign /ko-app/cosign /usr/local/bin/cosign
# cosign caches the Sigstore TUF root under $HOME/.sigstore.
ENV HOME=/tmp
USER 65532:65532
