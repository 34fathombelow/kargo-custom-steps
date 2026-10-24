# GitHub Actions

What each workflow does, when it runs, and what it needs. Everything runs on GitHub-hosted `ubuntu-latest` runners, and
every third-party action is pinned to a commit SHA (Renovate keeps the pins current).

| Workflow | Runs on | What it does |
|---|---|---|
| [check.yaml](check.yaml) | Every pull request; pushes to `main` | Lints the repository and runs every step's tests |
| [images.yaml](images.yaml) | Changes under `docker/`; by hand | Builds the step images, and on `main` publishes, attests and signs them |
| [release.yaml](release.yaml) | Pushing a `v*` tag | Publishes the custom steps as GitHub Release assets, pinned to image digests |

## check.yaml

Two jobs, both required to pass before merging:

- **lint**
  - `make lint` checks the repository conventions: steps use images this repo builds, at the tags their Dockerfiles
    declare; no template expressions inside scripts; every demo Stage and Warehouse is on one shard; every `uses:`
    names a real step; every step has a README and tests.
  - `make validate` checks every Kargo resource under `kargo/` against the Kargo CRD schemas, offline.
  - `helm lint` checks the small chart the `manifest-diff` demo renders.
- **test**
  - Builds every image in `docker/` locally, without pushing.
  - Starts the services some fixtures call (`make test-services`): a throwaway Postgres for `flyway-migrate` and a fake
    Argo CD metrics endpoint for `cluster-version`.
  - Runs every step's test fixtures (`make test`) with the local harness in [test/cmd/runstep](../../test/cmd/runstep).

It needs no secrets.

## images.yaml

Builds `docker/<tool>/` images for `linux/amd64` and `linux/arm64`.

| Trigger | Images built | Pushed? |
|---|---|---|
| Pull request touching `docker/` | Only the changed ones | No, build only |
| Push to `main` touching `docker/` | Only the changed ones | Yes |
| Run workflow (Actions tab), input `tool` | One tool, or `all` | Yes |

- **Where images go:** `ghcr.io/<repository owner>/<tool>`. Each image gets two tags: `<version>`, read from
  `ARG <TOOL>_VERSION` in its Dockerfile, and an immutable `<version>-<date>-<short sha>`.
- **Attestations:** each pushed image gets an SBOM and a build provenance attestation, and is signed keyless with
  cosign, using this workflow's GitHub identity.
- **Permissions:** `packages: write` to push to GHCR, and `id-token: write` for keyless signing. It uses the built-in
  `GITHUB_TOKEN` and needs no other secrets.
- **Visibility:** new GHCR packages start out private. Make them public, or Kargo's promotion pods can't pull them.

If you change a tool's version, update the image tag in every step that uses it in the same pull request; `make lint`
fails until you do.

## release.yaml

On a `v*` tag (e.g. `git tag v1.0.0 && git push origin v1.0.0`):

1. Resolves every step's image to its digest and writes a copy of each step with `image: <image>@sha256:...`.
2. Attaches them to a GitHub Release: one `<step>.yaml` per step, plus `kargo-custom-steps.yaml` with all of them.

The images must already be published (by images.yaml) before you tag. To use a release, copy the YAML into the git
repository your Argo CD syncs into Kargo.

Permissions: `contents: write`, to create the release.

## Renovate ([../renovate.json](../renovate.json))

Not a workflow, but it drives most version changes. It opens pull requests to bump:

- tool versions in `docker/<tool>/Dockerfile`, wherever an `ARG` has a `# renovate:` comment above it;
- the commit SHAs of the GitHub Actions in these workflows.

A tool bump changes that image's tag, so the pull request also needs the matching step YAML updated. It requires the
Renovate GitHub App to be installed on the repository.

## In a fork

Nothing here is tied to the original owner. Images go to `ghcr.io/<your account>`. Run `scripts/fork.sh` so the steps
and demos point at your registry and repository, then push. The first push to `main` builds every image, or you can
run **Images** by hand with `all`.
