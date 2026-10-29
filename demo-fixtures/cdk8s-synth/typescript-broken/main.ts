import { Construct } from 'constructs';
import { App, Chart, ChartProps } from 'cdk8s';
import { KubeDeployment, KubeService, IntOrString } from './imports/k8s';

// The Kargo cdk8s-synth step passes per-Stage settings as environment variables.
const environment = process.env.ENVIRONMENT ?? 'dev';
const image = process.env.IMAGE ?? 'ghcr.io/akuity/guestbook:v0.0.2';

export class Guestbook extends Chart {
  constructor(scope: Construct, id: string, props: ChartProps = {}) {
    super(scope, id, props);

    const labels = { app: 'guestbook-ui', environment };

    new KubeDeployment(this, 'deployment', {
      metadata: { name: 'guestbook-ui', labels },
      spec: {
        replicas: 'three', // intentional type error for the cdk8s-synth-blocked demo Stage
        selector: { matchLabels: { app: 'guestbook-ui' } },
        template: {
          metadata: { labels },
          spec: {
            containers: [{
              name: 'guestbook-ui',
              image,
              ports: [{ containerPort: 80 }],
            }],
          },
        },
      },
    });

    new KubeService(this, 'service', {
      metadata: { name: 'guestbook-ui', labels },
      spec: {
        selector: { app: 'guestbook-ui' },
        ports: [{ port: 80, targetPort: IntOrString.fromNumber(80) }],
      },
    });
  }
}

const app = new App();
new Guestbook(app, 'guestbook');
app.synth();