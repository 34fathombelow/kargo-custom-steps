# cluster-version fixtures

Fixtures 01-04 call a fake Kubernetes API ([fake-api.py](fake-api.py)) on docker network `runstep-k8s-api`.
Start it first:

```sh
make test-services          # starts it (and the flyway-migrate test database)
make test STEP=cluster-version
make test-services-down
```
