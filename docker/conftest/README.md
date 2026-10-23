# conftest

conftest (from `openpolicyagent/conftest`) plus `bash`, `git` and `jq`. `git` lets `conftest pull` fetch `git::` policy URLs. The upstream image has a shell but no `jq` and no `USER`; this image pins the version, adds `jq`, and runs as UID 65532.

Used by: `opa-conftest`.

## Build manually

```sh
docker build -t ghcr.io/34fathombelow/conftest:v0.71.1 docker/conftest
```
