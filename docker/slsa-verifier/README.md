# slsa-verifier

[slsa-verifier](https://github.com/slsa-framework/slsa-verifier) on Alpine with `bash` and `jq`. There is no official
image, so it is built from source with `go install` (the Go checksum database pins the module source), for linux/amd64
and linux/arm64. The `slsa-verify` step needs a shell and `jq` to shape the provenance into its output. `HOME=/tmp` so
slsa-verifier can cache the Sigstore TUF root, and write registry credentials, as UID 65532.

Used by: `slsa-verify`.

## Build manually

```sh
docker build -t ghcr.io/34fathombelow/slsa-verifier:v2.7.1 docker/slsa-verifier
```
