# Prerequisite: Argo CD apps for argocd-drift-check

The `argocd-drift-check-pass` and `argocd-drift-check-blocked` Stages in `custom-steps-release-insight` check two Argo CD Applications. Argo CD
creates them: [argocd/apps/prereqs-argocd-drift-check.yaml](../../argocd/apps/prereqs-argocd-drift-check.yaml) syncs
[applications.yaml](applications.yaml) into Argo CD. Nothing to apply by hand.

They must be in the Argo CD that the `custom-steps` shard's promotion pods reach through the `access-argocd` capability
(the Argo CD integrated with that Kargo agent). Their workloads deploy to the cluster registered in Argo CD as
`custom-steps-agent`.

| Application | Sync policy | State | Stage | Expected |
|---|---|---|---|---|
| `drift-demo-clean` | automated + self-heal | Synced, Healthy | `argocd-drift-check-pass` | Promotion succeeds |
| `drift-demo-unsynced` | manual, never synced | OutOfSync | `argocd-drift-check-blocked` | Promotion blocked: `... is OutOfSync: 2 resource(s) differ from git ...` |

Both deploy the public [argocd-example-apps guestbook](https://github.com/argoproj/argocd-example-apps/tree/master/guestbook).

To see drift on the healthy app too, use the Argo CD UI: open `drift-demo-clean`, turn off **Self Heal** under App
Details → Sync Policy, then edit a live resource (for example set the guestbook Deployment's replicas to 3). Promote
`argocd-drift-check-pass` again and it is blocked. Turn Self Heal back on to restore it.

If someone syncs `drift-demo-unsynced` from the UI, it becomes clean. Argo CD's next sync of the parent app does not undo
that, so delete the app in the UI and let the parent app recreate it unsynced.

If Argo CD runs in a namespace other than `argocd`, change `metadata.namespace` in applications.yaml and the `namespace`
var in `kargo/projects/release-insight/argocd-drift-check.yaml`.

To remove them, delete `argocd/apps/prereqs-argocd-drift-check.yaml` in your fork (Argo CD prunes both apps).
