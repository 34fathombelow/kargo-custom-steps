---
name: new-custom-step
description: Build a new Kargo custom promotion step (CustomPromotionStep) in this repo end to end - the step YAML and README, its image, test fixtures, the generated README sections and a demo project. Use when the user wants to add, create, scaffold or port a custom step, a promotion gate, or a tool that should run during a Kargo promotion.
---

# New custom step

Build the step the way every step in this repo is built, so `make check` passes and the demo works on a cluster.
Read `CLAUDE.md` first: its templating rules and decisions apply to every line you write.

## 1. Decide what to build

Ask, or infer from the request:
- **What the step does and when it blocks.** A gate exits non-zero to block; a step with side effects must be idempotent.
- **The tool** it runs, and whether an image in `docker/` already has it (`toolbox` has bash, jq, yq, curl, git).
- **Its group:** policy, supply-chain, manifest-quality, release-insight, data or rendering.
- **Its config keys and outputs.**

Before writing anything, check that Kargo doesn't already do it (built-in steps, Enterprise steps, Stage features such
as verification or promotion windows) and that the name doesn't clash with a built-in step. If Kargo already does it,
say so and stop: this repo only holds custom steps.

Pick a name in the repo's style: `<tool>-<verb or noun>` such as `trivy-image`, `cosign-verify`, `flyway-migrate`.

## 2. Write the step

Copy [templates/step.yaml](templates/step.yaml) to `kargo/custom-steps/<group>/<step>/<step>.yaml` and fill it in:
- Every config value goes through an `env` entry: `quote()` for simple tokens, `asJSON()` + `jq -j .` for secrets and
  free-form text. The script never contains `${{` or `${%`.
- Validate every required input first, with a message that names the fix ("config.path is required", "not found in the
  workdir (did an earlier git-clone step check it out?)").
- Decide from the tool's machine-readable output, not only its exit code. Write JSON to `/tmp/<step>.json`, including on
  failure, so the promotion shows why it stopped. Cap long lists.
- Never print secrets: the promotion message shows stdout and stderr.

## 3. The image (skip if an existing one has everything)

Copy [templates/Dockerfile](templates/Dockerfile) to `docker/<tool>/Dockerfile` and write a short `docker/<tool>/README.md`
(what the image adds and why; see `docker/kyverno/README.md`). Then:

```sh
make images TOOL=<tool>
docker run --rm --user 65532:65532 ghcr.io/34fathombelow/<tool>:<version> <tool> --version
```

The step's `image:` tag must equal the Dockerfile's `ARG <TOOL>_VERSION`. Prefer copying a static binary from the
upstream image into alpine, plus `bash ca-certificates jq`.

## 4. Fixtures and tests

Use the `test-custom-step` skill. At minimum: one passing and one failing fixture in `test/fixtures/<step>/`, plus one
fixture per input check. Run `make test STEP=<step>` until it passes.

## 5. Docs

1. Write `kargo/custom-steps/<group>/<step>/README.md` from [templates/README.md](templates/README.md). Fill in every
   row of the Prerequisites table.
2. `make step-images` adds the generated "Image" section.
3. Add the step to the `runs` map in `scripts/example-outputs.sh` (`[<step>]="Passes:<fixture> Blocked:<fixture>"`), then
   run `scripts/example-outputs.sh <step>` for the "Example output" section.
4. In the top-level `README.md`: a `#### <step>` section under its group in **Examples** (what it does, a one-line
   **Prerequisites:** summary, a short config example, links to the step and its README). Remove the step from
   **More custom steps to build** if it was listed.

## 6. Demo

In `kargo/projects/<group>/`, from [templates/promotion-task.yaml](templates/promotion-task.yaml) and
[templates/stages.yaml](templates/stages.yaml):
- A PromotionTask named `<step>.yaml` that clones this repo read-only and runs the step on files in `demo-fixtures/`.
- Stages named `<step>-<outcome>`: one that should pass (green), one that should be blocked (red).
- If the demo needs a new image or repo in its Freight, add a Warehouse subscription, pinned to one version.
- Workloads the demo needs go in `prereqs/<step>/` with an Argo CD app in `argocd/apps/`. Secrets are documented in
  `secrets/README.md`, never committed.
- A new group also needs `project.yaml` (sync-wave `-1`) and a row in the README's "Choosing which demos to run".

## 7. Finish

```sh
make check
```

Then report what was built, the commands that passed, and anything not covered: tests that need real credentials, and
the demo, which can only run after the image is published. Don't commit unless the user asks.
