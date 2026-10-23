# OPA plus a shell and jq, so opa-promotion-gate can build its input document.
# renovate: datasource=docker depName=openpolicyagent/opa
ARG OPA_VERSION=1.21.2
# renovate: datasource=docker depName=alpine
ARG ALPINE_VERSION=3.24.2

FROM openpolicyagent/opa:${OPA_VERSION}-static AS opa

FROM alpine:${ALPINE_VERSION}
LABEL org.opencontainers.image.source=https://github.com/34fathombelow/kargo-custom-steps \
      org.opencontainers.image.description="OPA for the Kargo opa-promotion-gate custom step"
RUN apk add --no-cache bash ca-certificates coreutils jq
COPY --from=opa /opa /usr/local/bin/opa
ENV HOME=/tmp
USER 65532:65532
