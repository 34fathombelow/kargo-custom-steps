# Node.js, the cdk8s CLI and Go for the Kargo cdk8s-synth-go custom step,
# which renders Go cdk8s apps (go mod download + cdk8s synth). The cdk8s Go
# bindings also need Node.js at run time (jsii).
# jq and yq turn the rendered manifests into the step's JSON output.
# renovate: datasource=npm depName=cdk8s-cli
ARG CDK8S_GO_VERSION=2.207.73
# renovate: datasource=docker depName=node versioning=node
ARG NODE_VERSION=26.12.1
# renovate: datasource=docker depName=alpine
ARG ALPINE_VERSION=3.24
# renovate: datasource=docker depName=golang
ARG GO_VERSION=1.27.2
# renovate: datasource=docker depName=mikefarah/yq
ARG YQ_VERSION=4.54.1

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS go
FROM mikefarah/yq:${YQ_VERSION} AS yq

FROM node:${NODE_VERSION}-alpine${ALPINE_VERSION}
ARG CDK8S_GO_VERSION
LABEL org.opencontainers.image.source=https://github.com/34fathombelow/kargo-custom-steps \
      org.opencontainers.image.description="cdk8s CLI with Node.js and Go for Go cdk8s apps, for the Kargo cdk8s-synth custom steps" \
      org.opencontainers.image.version=${CDK8S_GO_VERSION}
RUN apk add --no-cache bash ca-certificates jq && \
    npm install -g cdk8s-cli@${CDK8S_GO_VERSION} && \
    npm cache clean --force
COPY --from=go /usr/local/go /usr/local/go
COPY --from=yq /usr/bin/yq /usr/local/bin/yq
# Custom steps run as 65532: npm and Go write caches under a writable /tmp.
ENV HOME=/tmp \
    npm_config_cache=/tmp/.npm \
    npm_config_update_notifier=false \
    npm_config_fund=false \
    npm_config_audit=false \
    PATH=/usr/local/go/bin:$PATH \
    GOPATH=/tmp/go \
    GOCACHE=/tmp/.go-build \
    GOTOOLCHAIN=local \
    JSII_SILENCE_WARNING_UNTESTED_NODE_VERSION=1
USER 65532:65532
