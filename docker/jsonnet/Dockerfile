# go-jsonnet and jsonnet-bundler, built from source: no official multi-arch image.
# yq converts jsonnet's JSON output into readable YAML manifests.
# renovate: datasource=github-releases depName=google/go-jsonnet
ARG JSONNET_VERSION=v0.22.0
# renovate: datasource=github-releases depName=jsonnet-bundler/jsonnet-bundler
ARG JB_VERSION=v0.6.0
# renovate: datasource=docker depName=golang
ARG GO_VERSION=1.27.2
# renovate: datasource=docker depName=alpine
ARG ALPINE_VERSION=3.24.2
# renovate: datasource=docker depName=mikefarah/yq
ARG YQ_VERSION=4.54.1

FROM golang:${GO_VERSION}-alpine AS build
ARG JSONNET_VERSION
ARG JB_VERSION
RUN CGO_ENABLED=0 go install github.com/google/go-jsonnet/cmd/jsonnet@${JSONNET_VERSION} && \
    CGO_ENABLED=0 go install github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb@${JB_VERSION}

FROM mikefarah/yq:${YQ_VERSION} AS yq

FROM alpine:${ALPINE_VERSION}
LABEL org.opencontainers.image.source=https://github.com/34fathombelow/kargo-custom-steps \
      org.opencontainers.image.description="jsonnet + jb for the Kargo jsonnet-render custom step"
RUN apk add --no-cache bash ca-certificates git jq
COPY --from=build /go/bin/jsonnet /go/bin/jb /usr/local/bin/
COPY --from=yq /usr/bin/yq /usr/local/bin/yq
ENV HOME=/tmp
USER 65532:65532
