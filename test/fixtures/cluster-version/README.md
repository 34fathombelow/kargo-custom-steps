# cluster-version fixtures

The fixtures call a fake Argo CD application-controller metrics endpoint ([fake-api.py](fake-api.py)) on docker
network `runstep-k8s-api`, reachable as `http://argocd-metrics:8082`, the step's default URL. Start it first:

```sh
make test-services          # starts it (and the flyway-migrate test database)
make test STEP=cluster-version
make test-services-down
```
