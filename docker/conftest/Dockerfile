# conftest (OPA/Rego) plus a shell and jq, so opa-conftest can summarize results.
# renovate: datasource=docker depName=openpolicyagent/conftest
ARG CONFTEST_VERSION=v0.71.1
# renovate: datasource=docker depName=alpine
ARG ALPINE_VERSION=3.24.2

FROM openpolicyagent/conftest:${CONFTEST_VERSION} AS conftest

FROM alpine:${ALPINE_VERSION}
LABEL org.opencontainers.image.source=https://github.com/34fathombelow/kargo-custom-steps \
      org.opencontainers.image.description="conftest for the Kargo opa-conftest custom step"
RUN apk add --no-cache bash ca-certificates git jq
COPY --from=conftest /conftest /usr/local/bin/conftest
ENV HOME=/tmp
USER 65532:65532
