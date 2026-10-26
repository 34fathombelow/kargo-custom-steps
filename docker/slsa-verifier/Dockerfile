# slsa-verifier (SLSA build provenance verification), built from source: no
# official image. The Go checksum database pins the module source.
# renovate: datasource=github-releases depName=slsa-framework/slsa-verifier
ARG SLSA_VERIFIER_VERSION=v2.7.1
# renovate: datasource=docker depName=golang
ARG GO_VERSION=1.27.2
# renovate: datasource=docker depName=alpine
ARG ALPINE_VERSION=3.24.2

FROM golang:${GO_VERSION}-alpine AS build
ARG SLSA_VERIFIER_VERSION
RUN CGO_ENABLED=0 go install -trimpath -ldflags='-s -w' \
      github.com/slsa-framework/slsa-verifier/v2/cli/slsa-verifier@${SLSA_VERIFIER_VERSION}

FROM alpine:${ALPINE_VERSION}
LABEL org.opencontainers.image.source=https://github.com/34fathombelow/kargo-custom-steps \
      org.opencontainers.image.description="slsa-verifier for the Kargo slsa-verify custom step"
RUN apk add --no-cache bash ca-certificates jq
COPY --from=build /go/bin/slsa-verifier /usr/local/bin/slsa-verifier
# slsa-verifier caches the Sigstore TUF root under $HOME/.sigstore.
ENV HOME=/tmp
USER 65532:65532
