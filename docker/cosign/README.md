# cosign

[cosign](https://github.com/sigstore/cosign) on Alpine with `bash`, `jq` and `openssl`. The upstream image is distroless,
and the `cosign-verify` step needs a shell to shape its output and `openssl` to read signer identities from signature
certificates. `HOME=/tmp` so cosign can cache the Sigstore TUF root as UID 65532.

Used by: `cosign-verify`.

## Build manually

```sh
docker build -t ghcr.io/34fathombelow/cosign:v3.1.3 docker/cosign
```
