# scoring-apps-go

Contains applications demonstrating the methodology for calculating a confidence score based on received annotations.

# Build Notes

**NOTE: While generally applicable to all OS platforms, the specific instructions are relative to Linux (Ubuntu)**

This application has a dependency on the [alvarium-sdk-go](https://github.com/project-alvarium/alvarium-sdk-go) module.

Obtaining the `alvarium-sdk-go` via `go get` will allow you to build the applications and run tests.

Having done that, you will now be able to build the application using the `make build` command line.

If you wish to build Docker images of these services, you can do so via the `make docker` command line.

# Makefile execution

If you build the services from source, you can run them locally. There are several different permutations for the supporting services
which are triggered by the Makefile. Unless otherwise specified all services use MQTT for pub/sub, ArangoDB for persisting the Alvarium
DCF graph and MongoDB for the example "business database." The Mongo database is populated with example business data by the example applications
provided in [Go](https://github.com/project-alvarium/example-go) and [Java](https://github.com/project-alvarium/example-java).

- `make run` will start the services locally with a small delay between each.
- `make run_docker` uses the scripts/docker/docker-compose.yml file to bring up all of the services and their supporting applications.
  As indicated in the `make` argument this option also supports OPA for applying annotation weights by policy when calculating a score. You should enable the OPA server first via the scripts/policies/Dockerfile.
- `make run_opa` executes the services using all defaults except for OPA policy enablement. See above for how to start the OPA server via Docker.

# Scoring overview

Please see the `DCF_Scoring_Method_Proposed.pdf` document in the root of this repo for a complete flow diagram of how these services interact.

# Kubernetes deployment

- If you use a local docker registry for the K8s cluster, ensure that the scoring-apps docker images are already there.
- If you do not use a local docker registry for the K8s cluster, ensure that the scoring-apps docker images are composed created on each worker-node/host-node
- run 'helm install dcf scripts/alvarium_helm/ -n dcf --create-namespace'
