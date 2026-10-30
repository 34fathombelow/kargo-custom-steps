# syft

[Syft](https://github.com/anchore/syft), plus `bash` and `jq`, for generating SBOMs of container images straight from
their registry (no Docker daemon).

Used by: `sbom-generate`.

## Build manually

```sh
docker build -t ghcr.io/34fathombelow/syft:1.55.0 docker/syft
```
