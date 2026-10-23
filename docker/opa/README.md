# opa

OPA (from `openpolicyagent/opa`, static build) plus `bash`, `jq` and GNU `coreutils`. BusyBox `date` cannot parse RFC 3339 timestamps, which the step needs. The upstream image has no shell, so the step could not shape its output. This image adds one and runs as UID 65532.

Used by: `opa-promotion-gate`.

## Build manually

```sh
docker build -t ghcr.io/34fathombelow/opa:1.21.2 docker/opa
```
