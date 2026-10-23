# toolbox

General-purpose image for steps that are mostly shell: `bash`, `jq`, `yq`, `curl`, `git` and the GNU coreutils.

Used by: `release-diff`, `argocd-drift-check`.

## Build manually

```sh
docker build -t ghcr.io/34fathombelow/toolbox:1.2.0 docker/toolbox
```
