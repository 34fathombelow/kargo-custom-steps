<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/images/custom-promotion-steps-kargo-dark.png">
    <img src="docs/images/custom-promotion-steps-kargo-light.png" width="700"
         alt="Kargo kangaroo floating in space holding a ship's wheel above grey moon-rock stairs that lead to a prod flag, thinking: I asked for a promotion. They gave me stairs."
         title="The Warehouse said there'd be an elevator.">
  </picture>
</p>

# Kargo Custom Promotion Steps

## Overview

Custom Promotion Steps let Kargo administrators add new promotion steps to Kargo: run any container image, with any
command, as a step in a promotion. This repository is a catalogue of ready-to-use custom steps, organized into groups,
and a **runnable demo project for every group** so you can apply them, promote, and see each step pass and block.

Want to add one? See [More custom steps to build](#more-custom-steps-to-build) for ideas.

> Custom promotion steps require **pod-based promotions**, available on the [Akuity Platform](https://akuity.io).

## Why custom steps?

Kargo's built-in steps handle the most common GitOps promotion operations out of the box, with native integrations for
Argo CD, Git, Helm, Kustomize, OCI and more. Kargo Enterprise adds ServiceNow, Jira, GitHub, Terraform/OpenTofu,
approvals and messaging. But every organization has unique requirements: custom security gates, internal tooling,
or compliance checks that don't fit a standard mold. Custom steps make these possible.

Examples of when you would use a custom step:

* Policy-as-code gates: `kyverno`, `conftest`, `opa`
* Image vulnerability scanning, signature and provenance verification: `trivy`, `cosign`, `slsa-verifier`
* Manifest checks against the target cluster version: `pluto`
* Understanding a release: semantic manifest diffs, git history, Argo CD drift
* Organization-specific checks no built-in step covers
* Database schema migrations that travel with the release: `flyway`
* Rendering with tools Kargo doesn't ship: `jsonnet`
* Bespoke tools, scripts, and internal system integrations

## How it works

A `CustomPromotionStep` is a cluster-scoped resource that tells Kargo:

* the container `image` to run,
* the `command` to run in it,
* optional `env`, rendered from the step's `config`,
* optional `output` to collect when it finishes.

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: CustomPromotionStep
metadata:
  name: hello
spec:
  image: ghcr.io/34fathombelow/toolbox:1.2.0
  env:
  - name: GREETING
    value: ${{ asJSON(config.greeting ?? "Hello") }}
  command:
  - bash
  - -c
  - |
    greeting="$(jq -j . <<<"$GREETING")"
    echo "message=${greeting}, world" >> "$KARGO_OUTPUT"
```

Once registered, it can be used like any built-in step. Outputs are referenced by later steps through
`outputs['<as>'].<key>`:

```yaml
steps:
- uses: hello
  as: hello
  config:
    greeting: Bonjour
- uses: yaml-update
  config:
    path: ./src/values.yaml
    updates:
    - key: greeting
      value: ${{ outputs['hello'].message }}
```

The step runs in the promotion pod, in the same working directory as every other step, so it can read what
`git-clone` or `helm-template` produced and write files for later steps. Exit code `0` means the step succeeded;
anything else fails the promotion, which is how gates block.

## Prerequisites

Every step and demo needs:

* Kargo Enterprise with **pod-based promotions** (Akuity Platform), and this **Kargo agent**:

  | Name in Kargo (agent and shard) | Cluster | Needed for |
  |---|---|---|
  | `custom-steps` | Where the promotion pods run, so every custom step runs here | Everything |

  * The agent must run the full set of controllers for its shard: Warehouses (Freight discovery), Stages and Promotions.
  * Every demo Stage and Warehouse is pinned to the shard `custom-steps`. To use another one, run `make set-shard SHARD=<name>`.
* **Argo CD with these clusters registered** under these names. Argo CD is the only way anything here reaches a
  cluster: everything is GitOps, and nobody needs `kubectl` or access to the Kargo control plane.

  | Name in Argo CD | Cluster | Argo CD deploys there | Needed for |
  |---|---|---|---|
  | `kargo1` | The Kargo control plane | The custom steps and every demo project (`argocd/apps/kargo-custom-steps.yaml`) | Everything |
  | `custom-steps-agent` | Where the `custom-steps` Kargo agent runs (promotion pods run here) | The demo Postgres, and the two sample apps `argocd-drift-check` inspects | The `data` and `release-insight` demos |
  | `in-cluster` | Argo CD itself | The root app-of-apps and the two sample Applications | Setup, and the `release-insight` demo |

  * Different names? Run `scripts/fork.sh --argocd-cluster <kargo name> --agent-cluster <agent name>` ([Forking](#forking)).
  * On the Akuity Platform, enable **Declarative Management** on the Argo CD instance, so `in-cluster` is available and
    apps can create other apps.
  * For `argocd-drift-check`, this must be the Argo CD integrated with the `custom-steps` Kargo agent: the step reads
    Applications through that integration.
* Network egress from the agent cluster to `ghcr.io` (step images) and `github.com` (the demos clone this repository).
  Some steps reach more hosts, such as `mirror.gcr.io` (falling back to `ghcr.io`) for Trivy's vulnerability DBs and Sigstore for cosign and slsa-verifier.

Some steps need more: a database, an Argo CD integration, or egress to other hosts. Each step's README lists its own
under **Prerequisites** (cluster add-ons and which cluster, reachable services, accounts, Kargo capabilities, egress,
and inputs from earlier steps), and [Choosing which demos to run](#choosing-which-demos-to-run) lists the extra setup
each demo needs.

## Running the demos

Everything is GitOps: Argo CD syncs this repository, and the only manual steps happen in the Argo CD and Kargo UIs.

To run everything as-is, point Argo CD at this repository. To customize anything (cluster names, which demos run,
which prerequisites deploy), **fork it first**: Argo CD syncs from git, so changes only take effect in your fork.
[Forking](#forking) covers `scripts/fork.sh`, which points every app at your fork.

1. **Check the clusters in Argo CD.** `kargo1` and `custom-steps-agent` must be registered, and `in-cluster` available
   (see [Prerequisites](#prerequisites)). If your names differ, fork and run `scripts/fork.sh` with them first.
2. **Create the root app in the Argo CD UI:** + New App → Edit as YAML → paste [argocd/root.yaml](argocd/root.yaml) →
   Create. It syncs every app in [argocd/apps/](argocd/apps):

   | App | Syncs | To |
   |---|---|---|
   | `kargo-custom-steps` | every custom step and every group's demo project (`custom-steps-<group>`) from `kargo/` | `kargo1` |
   | `kargo-custom-steps-prereq-postgres` | the demo Postgres for `flyway-migrate` ([prereqs/flyway-migrate](prereqs/flyway-migrate)) | `custom-steps-agent` |
   | `kargo-custom-steps-prereq-drift-apps` | the two sample apps `argocd-drift-check` inspects ([prereqs/argocd-drift-check](prereqs/argocd-drift-check)) | `in-cluster` (Argo CD itself) |

   To run only some demos, see [Choosing which demos to run](#choosing-which-demos-to-run). To skip a prerequisite,
   delete its file from `argocd/apps/` in your fork.
3. **Create the one secret the `data` demo needs** in the Kargo UI: `flyway-db` ([secrets/README.md](secrets/README.md)).
4. **Promote in the Kargo UI.** Open a `custom-steps-<group>` project and promote the newest Freight into each Stage.
   Green Stages should succeed and red ones should be blocked (yellow marks the `data` demo's rollback case, which
   succeeds as a no-op). Each step's output is shown on the promotion. Two demos run in order:
   promote `flyway-migrate-apply` before `flyway-migrate-rollback`, and `release-diff-dev` before
   `release-diff-staging` (which takes its Freight from `release-diff-dev`).

### Choosing which demos to run

This is a change in git, so it needs a [fork](#forking). Each group is two folders under `kargo/`: its custom steps
(`custom-steps/<group>/`) and its demo project (`projects/<group>/`). Select groups in [argocd/apps/kargo-custom-steps.yaml](argocd/apps/kargo-custom-steps.yaml) with
`directory.include` or `directory.exclude`, using two patterns per group:

```yaml
spec:
  source:
    path: kargo
    directory:
      recurse: true
      # Only the policy and supply-chain groups:
      include: '{custom-steps/policy/*,projects/policy/*,custom-steps/supply-chain/*,projects/supply-chain/*}'
      # ...or everything except the data group:
      # exclude: '{custom-steps/data/*,projects/data/*}'
```

| Group | Patterns | Needs extra setup |
|---|---|---|
| policy | `custom-steps/policy/*`, `projects/policy/*` | no |
| supply-chain | `custom-steps/supply-chain/*`, `projects/supply-chain/*` | no |
| manifest-quality | `custom-steps/manifest-quality/*`, `projects/manifest-quality/*` | no |
| rendering | `custom-steps/rendering/*`, `projects/rendering/*` | no |
| release-insight | `custom-steps/release-insight/*`, `projects/release-insight/*` | Two sample Argo CD apps, synced automatically ([prereqs](prereqs/argocd-drift-check)) |
| data | `custom-steps/data/*`, `projects/data/*` | Postgres (synced automatically, [prereqs](prereqs/flyway-migrate)) and the `flyway-db` secret ([secret](secrets/README.md)) |

Patterns are matched against paths relative to `kargo/`, and `*` also matches across folders, so
`custom-steps/policy/*` covers every step in the group. Removing a group from the selection prunes its steps and its
Project, and Kargo then deletes the Project's namespace.

A good first run is the groups that need no extra setup: policy, supply-chain, manifest-quality and rendering.

## Examples

Each step's README has everything about it: what it does, its prerequisites, config and outputs, a walkthrough of its
demo Stages, and an **Example output** section with the real output of a passing and a blocked run.

| Group | What it's for | Custom steps | Demo project |
|---|---|---|---|
| [Policy](kargo/custom-steps/policy) | Policy-as-code gates: check manifests, or the promotion itself, against Kyverno or OPA policies from git | [`kyverno-apply`](kargo/custom-steps/policy/kyverno-apply/README.md), [`opa-conftest`](kargo/custom-steps/policy/opa-conftest/README.md), [`opa-promotion-gate`](kargo/custom-steps/policy/opa-promotion-gate/README.md) | [`custom-steps-policy`](kargo/projects/policy) |
| [Supply chain](kargo/custom-steps/supply-chain) | Is this image safe, and is it really ours? Vulnerabilities, signatures and build provenance, checked at promotion time | [`trivy-image`](kargo/custom-steps/supply-chain/trivy-image/README.md), [`cosign-verify`](kargo/custom-steps/supply-chain/cosign-verify/README.md), [`slsa-verify`](kargo/custom-steps/supply-chain/slsa-verify/README.md) | [`custom-steps-supply-chain`](kargo/projects/supply-chain) |
| [Manifest quality](kargo/custom-steps/manifest-quality) | Will these manifests work on the target cluster? Checks rendered manifests against its Kubernetes version | [`deprecated-apis`](kargo/custom-steps/manifest-quality/deprecated-apis/README.md), [`cluster-version`](kargo/custom-steps/manifest-quality/cluster-version/README.md) | [`custom-steps-manifest-quality`](kargo/projects/manifest-quality) |
| [Release insight](kargo/custom-steps/release-insight) | What exactly is changing? Manifest diffs, the commits a promotion brings in, and drift from git | [`manifest-diff`](kargo/custom-steps/release-insight/manifest-diff/README.md), [`release-diff`](kargo/custom-steps/release-insight/release-diff/README.md), [`argocd-drift-check`](kargo/custom-steps/release-insight/argocd-drift-check/README.md) | [`custom-steps-release-insight`](kargo/projects/release-insight) |
| [Data](kargo/custom-steps/data) | Move the database with the release: schema migrations applied during promotion, safe to retry and roll back | [`flyway-migrate`](kargo/custom-steps/data/flyway-migrate/README.md) | [`custom-steps-data`](kargo/projects/data) |
| [Rendering](kargo/custom-steps/rendering) | Render manifests with tools Kargo doesn't ship, into the workdir for later steps | [`jsonnet-render`](kargo/custom-steps/rendering/jsonnet-render/README.md) | [`custom-steps-rendering`](kargo/projects/rendering) |

## Before you write a custom step

Check whether Kargo already does it. Kargo ships many
[promotion steps](https://docs.kargo.io/user-guide/reference-docs/promotion-steps/) (`http` alone covers most
"call this API" or "wait until this endpoint says yes" cases), Stages have native gating such as `requiredSoakTime`
and `availabilityStrategy`, and Kargo Enterprise adds steps for ServiceNow, Jira, GitHub, Terraform/OpenTofu,
approvals and messaging, plus promotion windows for change freezes and release slots. Write a custom step when the work needs a tool or logic those can't express.

## Writing your own

The examples here follow a few rules that come from how the executor actually runs a step. They are worth knowing
before you write one:

1. **Pass config through `env`, never into the script body.** `${{ config.x }}` inside a `bash -c` script breaks on
   the first `'` or `"` in a value, is a shell-injection vector, and puts secrets on the process command line. Put
   each value in an `env` entry and read `"$VAR"` in a static script.
2. **Never pass a value raw.** Rendered values are re-parsed: `"1.30"` becomes `1.3`, `"007"` becomes `7`, the short
   git SHA `1234567` becomes `1.234567e+06`, and maps arrive in Go syntax (`map[a:b]`). `quote()` fixes simple tokens
   like versions and SHAs, but it applies Go escaping, so `pa"ss` arrives as `pa\"ss`.
3. **For secrets, free-form text and structures, use `asJSON()` as the entire `env` value** and decode it in the
   script: `jq -j .` for a string, `jq -c .` for a list or map. It is the only form that passes every value intact.
4. **There is no shell unless you ask for one.** `command[0]` is executed directly and the image's entrypoint is
   ignored, so use `bash -c` (or `sh -c`) explicitly.
5. **The step runs as UID 65532** with all Linux capabilities dropped. Images that write to `$HOME` or a cache
   directory need `HOME=/tmp` or a custom image. Every image in [docker/](docker) is built for this.
6. **Exit code is the only status.** Exit 0 succeeds, anything else fails. There is no "running" state, so a step
   that waits must loop inside its own command.
7. **Output.** `output.source` can be the `$KARGO_OUTPUT` file (`key=value`), stdout, stderr, or a file. A `File`
   path must be absolute (e.g. `/tmp/result.json`). A `transform` reshapes it, the result is capped at 256 KiB, and a
   `links` list of `{url, label}` renders as links in the UI. Output is recorded even when the step fails, so a gate
   can say *why* it blocked.
8. **Fail with a message that names the fix.** "`config.password` is empty: did you create the flyway-db secret?"
   beats a stack trace from the tool.
9. **Assume retries.** A step can run more than once. Make side effects idempotent.

## Testing steps locally

You don't need a Kargo instance to develop a step. [test/cmd/runstep](test/cmd/runstep) runs a step the way the
executor does. It renders `command` and `env` with Kargo's own template engine (so the re-parsing rules above
reproduce exactly), runs the image as UID 65532 with all capabilities dropped, and collects the output.

```sh
make images                       # build the step images locally
make test-services                # start the services some fixtures call (a throwaway Postgres
                                  # for flyway-migrate, a fake Argo CD metrics endpoint for cluster-version)
make test STEP=deprecated-apis    # run one step's fixtures (test/fixtures/<step>/*.yaml)
make test V=1                     # all steps, verbose
make lint                         # repository conventions
make validate                     # Kargo CRD schemas (offline)
make check                        # lint + validate + test, what CI runs
make test-services-down           # remove the test services
```

A fixture is the step's `config` and `ctx`, the files to place in the working directory, and the expected status and
outputs:

```yaml
description: at 1.33 the v2beta2 HPA is removed, so the promotion is blocked
config:
  path: ./out
  k8sVersion: "1.33"
files:
  out: demo-fixtures/deprecated-apis/manifests   # stands in for an earlier helm-template step
expect:
  status: Failed
  output:
    passed: false
    counts: {removed: 1, deprecated: 0}
```

## Contributing

Contributions are welcome: a new step, a fix, or a better demo. Pick an idea below or bring your own,
then follow [Adding a step](#adding-a-step).

Using [Claude Code](https://code.claude.com)? [CLAUDE.md](CLAUDE.md) gives it the repo's rules and the reasons behind
them, and two skills walk it through the work: `new-custom-step` (the whole checklist below) and `test-custom-step`
(fixtures, the local harness and troubleshooting), in [.claude/skills/](.claude/skills).

### More custom steps to build

Ideas for custom steps that would fit here, by group. Contributions welcome: see [Adding a step](#adding-a-step).

| Group | Step | What it would do |
|---|---|---|
| policy | `gatekeeper-gator` | Test manifests offline against existing OPA Gatekeeper ConstraintTemplates and Constraints with `gator` |
| policy | `kyverno-apply` `cluster` mode (enhancement) | Evaluate against the Kyverno policies installed in the target cluster, so the gate matches admission exactly |
| supply-chain | `grype-image` | The `trivy-image` gate with [Grype](https://github.com/anchore/grype) |
| supply-chain | `extract-images` | List every image in rendered manifests, to scan or verify each one |
| supply-chain | `sbom-generate` | Write an SBOM ([Syft](https://github.com/anchore/syft)) next to the manifests |
| supply-chain | `notation-verify` | Verify [Notation](https://notaryproject.dev) signatures |
| manifest-quality | `kubeconform` | Schema-validate manifests, including CRDs, for the target Kubernetes version |
| manifest-quality | `kube-linter`, `polaris-audit` | Best-practice linting (resource limits, probes, root containers) with a score or issue count |
| manifest-quality | `checkov-scan` | Security misconfiguration scanning for Kubernetes, Helm and Terraform |
| release-insight | `release-notes` | A changelog from conventional commits ([git-cliff](https://git-cliff.org)) between two revisions |
| release-insight | `pulumi-preview` | A `pulumi preview` summary as an output, for Pulumi-managed infrastructure |
| release-insight | `infracost-diff` | The monthly cost change of a Terraform change ([Infracost](https://www.infracost.io)) |
| data | `atlas-lint` | Flag destructive schema changes before they ship ([Atlas](https://atlasgo.io)) |
| data | `liquibase-update` | `flyway-migrate` for Liquibase changelogs |
| rendering | `kcl-render`, `cue-export`, `timoni-build` | Render manifests with [KCL](https://www.kcl-lang.io), [CUE](https://cuelang.org) or [Timoni](https://timoni.sh) |
| rendering | `cdk8s-synth` | Synthesize manifests from a [cdk8s](https://cdk8s.io) app (TypeScript, Python, Go or Java) |
| rendering | `helmfile-template` | Render a [Helmfile](https://helmfile.readthedocs.io) (several releases and environments) with `helmfile template` |
| rendering | `envsubst-render` | Render templates with [gomplate](https://gomplate.ca) or `envsubst` |
| rendering | `sops-decrypt` | Decrypt [SOPS](https://getsops.io) files (age or KMS) during rendering |
| rendering | `dockerfile-update` | Set a Dockerfile's `ARG` defaults and `FROM` tags/digests from Freight (e.g. a new base image), in the workdir. Kargo's `yaml-update` and image steps don't handle Dockerfiles. The output shows the diff |
| release-insight | `sentry-release` | Create a [Sentry](https://sentry.io) release and associate its commits with `sentry-cli` |

Before adding one, check that Kargo doesn't already do it (see [Before you write a custom step](#before-you-write-a-custom-step)).

### Adding a step

Check [Before you write a custom step](#before-you-write-a-custom-step) first, then work through this list.
`make lint` checks the items marked *(lint)*.

Where one step's files live, using `kyverno-apply` in the `policy` group:

```text
kargo/
├── custom-steps/policy/kyverno-apply/
│   ├── kyverno-apply.yaml     # the CustomPromotionStep
│   └── README.md
└── projects/policy/           # the group's demo project
    ├── project.yaml
    ├── warehouse.yaml
    ├── stages.yaml            # kyverno-apply-pass, kyverno-apply-blocked, ...
    └── kyverno-apply.yaml     # the step's PromotionTask
docker/kyverno/                # the step's image
├── Dockerfile
└── README.md
test/fixtures/kyverno-apply/   # harness fixtures: at least one passing, one failing
```

**The step**
- [ ] `kargo/custom-steps/<group>/<step>/<step>.yaml` *(lint)*. Its name must not clash with a built-in step *(lint)*.
- [ ] A `README.md` next to it, following [kyverno-apply](kargo/custom-steps/policy/kyverno-apply/README.md)'s structure:
  Prerequisites, Config, Outputs, Example, Try it, Notes. The step fails with a message naming any missing prerequisite.

**The image** (skip if an existing image in `docker/` has what you need)
- [ ] `docker/<tool>/Dockerfile`: `FROM` a pinned upstream image, `ARG <TOOL>_VERSION=<version>` with a `# renovate:` comment
  above it, the `org.opencontainers.image.source` and `.description` labels, and `USER 65532:65532` (plus `HOME=/tmp` if
  the tool writes to its home directory). The step's `image:` tag must match the `ARG` *(lint)*.
- [ ] `docker/<tool>/README.md`: what the image adds and why.
- [ ] `make images TOOL=<tool>` to build it locally. CI publishes it only after merge, so the demo can't run on a cluster
  before then.

**Tests** (see [Testing steps locally](#testing-steps-locally))
- [ ] At least one passing and one failing fixture in `test/fixtures/<step>/` *(lint)*.
- [ ] If the fixtures call a service (a database, an API), add it to `make test-services` and start it first.
- [ ] `make test STEP=<step>` until it passes.

**Generated docs**
- [ ] `make step-images` adds the README's "Image" section *(lint)*.
- [ ] Add the step to the `runs` map in `scripts/example-outputs.sh`, then run `scripts/example-outputs.sh <step>` to add
  its "Example output" section *(lint)*.

**Top-level README**
- [ ] The step, linked to its README, in its group's row of the [Examples](#examples) table (a new group gets a row).
- [ ] Any extra demo setup in [Choosing which demos to run](#choosing-which-demos-to-run).
- [ ] Remove it from [More custom steps to build](#more-custom-steps-to-build) if it was listed there.

**The demo** in `kargo/projects/<group>/`
- [ ] A PromotionTask named `<step>.yaml`, and Stages named `<step>-<outcome>` with a `kargo.akuity.io/color` of green
  (should pass) or red (should be blocked).
- [ ] `spec.shard` and the `kargo.akuity.io/shard` label on every Stage and Warehouse *(lint)*.
- [ ] A Warehouse subscription if the demo needs a new image or repository.
- [ ] Workloads the demo needs go in `prereqs/<step>/` with an app in `argocd/apps/`. Secrets are documented in
  [secrets/README.md](secrets/README.md) and created in the Kargo UI, never committed (a Secret under `kargo/` fails lint).
- [ ] A new group also needs `project.yaml` (sync-wave `-1`) and a row in
  [Choosing which demos to run](#choosing-which-demos-to-run).

**Before opening a PR**
- [ ] `make check` (lint, schema validation and every fixture). CI runs the same checks on the PR.

## Repository layout

| Path | What's there |
|---|---|
| [`kargo/custom-steps/<group>/<step>/`](kargo/custom-steps) | Each custom step: the `CustomPromotionStep` and its README |
| [`kargo/projects/<group>/`](kargo/projects) | Each group's demo project: Project, Warehouse, PromotionTasks, Stages |
| [`argocd/`](argocd) | `root.yaml` (created once in the Argo CD UI) and the apps it syncs, in `argocd/apps/` |
| [`prereqs/`](prereqs) | Services some demos need on the agent cluster, synced by Argo CD (never into Kargo) |
| [`secrets/`](secrets) | Which secrets to create in the Kargo UI. Secrets are never stored in git |
| [`docker/<tool>/`](docker) | Step images, built and published to `ghcr.io/34fathombelow` by GitHub Actions |
| [`demo-fixtures/`](demo-fixtures) | Inputs the demos and tests read: manifests, Kyverno and OPA policies (`policies/`), migrations, jsonnet |
| [`test/`](test) | The local step harness (`cmd/runstep`), the lint tool (`cmd/lint`) and each step's fixtures |
| [`scripts/`](scripts) | `fork.sh`, and the generators for the README sections (`example-outputs.sh`, `step-images.sh`) |
| [`.github/workflows/`](.github/workflows) | CI: checks, image builds, releases ([what each does](.github/workflows/README.md)) |
| [`docs/images/`](docs/images) | The logo |
| [`CLAUDE.md`](CLAUDE.md), [`.claude/skills/`](.claude/skills) | Context and skills for Claude Code: the repo's rules, decisions, and how to build and test a step |
| [`Makefile`](Makefile) | Every command above: `make help` lists them |

## Forking

To change steps or build the images yourself, fork the repository and retarget it at your fork:

```sh
scripts/fork.sh                                   # infers <owner>/<repo> from `git remote get-url origin`
scripts/fork.sh --repo acme/kargo-custom-steps --registry ghcr.io/acme --shard my-shard \
  --argocd-cluster my-kargo --agent-cluster my-agent
scripts/fork.sh --dry-run                         # show what would change
```

It rewrites every reference to the upstream repository and image registry: repository URLs in the demo projects and
the Argo CD apps, image references in every step, the Makefile's `REGISTRY`, and the Go module path. Optionally
it also changes the Kargo shard the demos use and the Argo CD cluster names of your Kargo control plane and agent cluster,
in the demo projects and Argo CD apps, the docs, step messages, and the templates the Claude Code skill uses for new
steps. Generated example outputs and test data keep the original names. You can re-run it.

Then build the images. Either push to GitHub and let the **Images** workflow publish every `docker/<tool>` image to your
registry (Actions → Images → Run workflow → `all` to build them all by hand), or build and push from your machine with
`make push` (multi-arch; needs `docker login`). GHCR packages start out private: make them public, or promotion pods
can't pull them.
