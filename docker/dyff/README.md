# dyff

[dyff](https://github.com/homeport/dyff) (semantic YAML/Kubernetes diff) plus `bash` and `jq`. There is no official
multi-arch dyff image, so it is built from source with `go install`.

Used by: `manifest-diff`.

## Build manually

```sh
docker build -t ghcr.io/34fathombelow/dyff:v1.12.0 docker/dyff
```
