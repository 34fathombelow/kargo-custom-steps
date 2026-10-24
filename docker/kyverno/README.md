# kyverno

Kyverno CLI (`kyverno`, from `ghcr.io/kyverno/kyverno-cli`) plus `bash` and `jq`. The upstream image has no shell, so the step could not shape its output. This image adds one and runs as UID 65532.

Used by: `kyverno-apply`.

## Build manually

```sh
docker build -t ghcr.io/34fathombelow/kyverno:v1.19.1 docker/kyverno
```
