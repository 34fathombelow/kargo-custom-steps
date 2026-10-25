# Promotion policy for the opa-promotion-gate step.
#
# The step evaluates data.kargo.promotion and expects an object with:
#   allow   - bool, whether the promotion may proceed
#   reasons - list of strings explaining every denial
#
# input (built by the step):
#   project, stage, promotion, actor, rollback, freight {name, alias, origin},
#   timestamp (RFC 3339, UTC), weekday ("Mon".."Sun"), hour (0-23, UTC),
#   plus every key of the step's config.input (e.g. environment, change_ticket).
package kargo.promotion

import rego.v1

# Who may promote to production.
release_managers := {
	"release-manager@example.com",
	"oncall-lead@example.com",
}

production if input.environment == "prod"

production if startswith(input.stage, "prod")

deny contains msg if {
	production
	not release_managers[input.actor]
	msg := sprintf("%s may not promote to production; only release managers can (%s)", [input.actor, concat(", ", sort(release_managers))])
}

deny contains msg if {
	production
	input.weekday == "Fri"
	input.hour >= 14
	msg := "no production promotions after 14:00 UTC on Fridays"
}

has_change_ticket if {
	is_string(input.change_ticket)
	input.change_ticket != ""
}

deny contains msg if {
	production
	not has_change_ticket
	msg := "production promotions need a change ticket (config.input.change_ticket)"
}

default allow := false

# Rollbacks are always allowed, so a bad release can be backed out.
allow if input.rollback

allow if count(deny) == 0

reasons := [] if input.rollback

reasons := sort(deny) if not input.rollback
