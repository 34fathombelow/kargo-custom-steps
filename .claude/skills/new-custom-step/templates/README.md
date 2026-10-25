# <step>

<What the step does, when it blocks, and why you'd use it in a promotion. Two or three sentences.>

## Prerequisites

Beyond the [common prerequisites](../../../../README.md#prerequisites):

| | |
|---|---|
| Cluster add-ons | <None, or what must be installed, and on which cluster> |
| Reachable services | <None, or what the promotion pod must reach> |
| Accounts / credentials | <None, or which secret, created in the Kargo UI> |
| Kargo capabilities | <None, or e.g. `access-argocd` (declared by the step)> |
| Network egress | <None, or the hosts the step contacts> |
| Inputs | <Files earlier steps put in the workdir, e.g. `./out` from `helm-template`> |

<!-- `make step-images` inserts the "Image" section here. -->

## Config

| Key | Type | Required | Default | Description |
|---|---|---|---|---|
| `path` | string | yes | — | <...> |

## Outputs

| Key | Type | Description |
|---|---|---|
| `passed` | bool | Whether the promotion may continue |
| `error` | string | Only when the step fails before deciding: why |

## Example

```yaml
steps:
- uses: <step>
  as: <alias>
  config:
    path: ./out
```

<!-- `scripts/example-outputs.sh <step>` inserts the "Example output" section here. -->

## Try it

Demo project `custom-steps-<group>` ([kargo/projects/<group>](../../../projects/<group>)):

| Stage | What it does | Expected |
|---|---|---|
| `<step>-pass` | <...> | Promotion succeeds |
| `<step>-blocked` | <...> | Promotion blocked: `<message>` |

## Notes

- <Idempotency, retries, limits, anything surprising about the tool.>
- Tested: harness fixtures in [`test/fixtures/<step>`](../../../../test/fixtures/<step>).
