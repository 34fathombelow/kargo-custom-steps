// Small library used by the jsonnet-render demo (passed with jpath).
{
  labels(name):: { 'app.kubernetes.io/name': name },

  deployment(name, image, replicas):: {
    apiVersion: 'apps/v1',
    kind: 'Deployment',
    metadata: { name: name, labels: $.labels(name) },
    spec: {
      replicas: replicas,
      selector: { matchLabels: $.labels(name) },
      template: {
        metadata: { labels: $.labels(name) },
        spec: {
          containers: [{
            name: name,
            image: image,
            ports: [{ containerPort: 3000 }],
            resources: { requests: { cpu: '50m', memory: '64Mi' }, limits: { memory: '128Mi' } },
          }],
        },
      },
    },
  },

  service(name):: {
    apiVersion: 'v1',
    kind: 'Service',
    metadata: { name: name, labels: $.labels(name) },
    spec: { selector: $.labels(name), ports: [{ port: 80, targetPort: 3000 }] },
  },
}
