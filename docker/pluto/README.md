# pluto

[Fairwinds pluto](https://github.com/FairwindsOps/pluto) for finding deprecated and removed Kubernetes APIs, with
`bash` and `jq`. There is no official pluto image, so the checksum-verified release binary is installed on Alpine
(amd64 and arm64).

Used by: `deprecated-apis`.

## Build manually

```sh
docker build -t ghcr.io/34fathombelow/pluto:v5.24.4 docker/pluto
```
