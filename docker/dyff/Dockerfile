# homeport dyff (semantic YAML diff), built from source: no official multi-arch image.
# renovate: datasource=github-releases depName=homeport/dyff
ARG DYFF_VERSION=v1.12.0
# renovate: datasource=docker depName=golang
ARG GO_VERSION=1.27.2
# renovate: datasource=docker depName=alpine
ARG ALPINE_VERSION=3.24.2

FROM golang:${GO_VERSION}-alpine AS build
ARG DYFF_VERSION
RUN CGO_ENABLED=0 go install github.com/homeport/dyff/cmd/dyff@${DYFF_VERSION}

FROM alpine:${ALPINE_VERSION}
LABEL org.opencontainers.image.source=https://github.com/34fathombelow/kargo-custom-steps \
      org.opencontainers.image.description="dyff for the Kargo manifest-diff custom step"
RUN apk add --no-cache bash ca-certificates jq
COPY --from=build /go/bin/dyff /usr/local/bin/dyff
ENV HOME=/tmp
USER 65532:65532
