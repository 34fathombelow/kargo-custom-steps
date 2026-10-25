# Conftest policies for Kubernetes workloads, used by the opa-conftest step.
# `deny` rules fail the promotion; `warn` rules are reported but do not block
# (unless the step's failOnWarn is set).
package main

import rego.v1

workload_kinds := {"Deployment", "StatefulSet", "DaemonSet", "Job"}

containers contains c if {
	workload_kinds[input.kind]
	some c in input.spec.template.spec.containers
}

name := sprintf("%s/%s", [input.kind, input.metadata.name])

deny contains msg if {
	some c in containers
	c.securityContext.privileged == true
	msg := sprintf("%s: container %q must not be privileged", [name, c.name])
}

deny contains msg if {
	some c in containers
	c.securityContext.runAsUser == 0
	msg := sprintf("%s: container %q must not run as root (runAsUser: 0)", [name, c.name])
}

deny contains msg if {
	some c in containers
	not c.resources.limits.memory
	msg := sprintf("%s: container %q has no memory limit", [name, c.name])
}

warn contains msg if {
	workload_kinds[input.kind]
	not input.spec.template.spec.securityContext.runAsNonRoot
	msg := sprintf("%s: set spec.template.spec.securityContext.runAsNonRoot: true", [name])
}

warn contains msg if {
	input.kind == "Deployment"
	input.spec.replicas < 2
	msg := sprintf("%s: fewer than 2 replicas means downtime during node maintenance", [name])
}
