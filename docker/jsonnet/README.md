# jsonnet

[go-jsonnet](https://github.com/google/go-jsonnet) and [jsonnet-bundler](https://github.com/jsonnet-bundler/jsonnet-bundler)
(`jb`), built from source because there is no official multi-arch image. Also includes `bash`, `git` (for `jb install`),
`jq` and `yq` (to turn jsonnet's JSON into YAML manifests).

Used by: `jsonnet-render`.

## Build manually

```sh
docker build -t ghcr.io/34fathombelow/jsonnet:v0.22.0 docker/jsonnet
```
