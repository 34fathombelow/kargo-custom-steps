// Entry point for the jsonnet-render demo.
//   ext var `image`        - full image reference, required (set from the Freight)
//   top-level arg replicas - optional, defaults to "1"
local app = import 'app.libsonnet';

function(replicas='1') {
  deployment: app.deployment('guestbook', std.extVar('image'), std.parseInt(replicas)),
  service: app.service('guestbook'),
}
