# Fairwinds pluto (deprecated/removed Kubernetes API detection). There is no
# official image, so the release binary is downloaded and checksum-verified.
# renovate: datasource=github-releases depName=FairwindsOps/pluto
ARG PLUTO_VERSION=v5.24.4
# renovate: datasource=docker depName=alpine
ARG ALPINE_VERSION=3.24.2

FROM alpine:${ALPINE_VERSION}
ARG PLUTO_VERSION
ARG TARGETARCH
LABEL org.opencontainers.image.source=https://github.com/34fathombelow/kargo-custom-steps \
      org.opencontainers.image.description="pluto for the Kargo deprecated-apis custom step"
RUN apk add --no-cache bash ca-certificates jq && \
    v="${PLUTO_VERSION#v}" && arch="${TARGETARCH:-amd64}" && \
    base="https://github.com/FairwindsOps/pluto/releases/download/${PLUTO_VERSION}" && \
    cd /tmp && \
    wget -q "${base}/pluto_${v}_linux_${arch}.tar.gz" "${base}/checksums.txt" && \
    grep " pluto_${v}_linux_${arch}.tar.gz\$" checksums.txt | sha256sum -c - && \
    tar -xzf "pluto_${v}_linux_${arch}.tar.gz" pluto && \
    install -m 0755 pluto /usr/local/bin/pluto && \
    rm -f /tmp/*
ENV HOME=/tmp
USER 65532:65532
