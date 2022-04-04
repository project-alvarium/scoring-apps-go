# scoring-apps-go
Contains applications demonstrating the methodology for calculating a confidence score based on received annotations.

# Build Notes

**NOTE: While generally applicable to all OS platforms, the specific instructions are relative to Linux (Ubuntu)**

This application has a dependency on the [alvarium-sdk-go](https://github.com/project-alvarium/alvarium-sdk-go) module
and through that a dependency on the [IOTA Streams C bindings](https://github.com/iotaledger/streams/tree/develop/bindings/c).

The SDK contains a pre-built artifact of the [C bindings](https://github.com/project-alvarium/alvarium-sdk-go/blob/main/internal/iota/include/libiota_streams_c.so)
in its source tree that was built on Ubuntu 20.04. Obtaining the `alvarium-sdk-go` via `go get` will allow you to build the
applications and run tests. However you will need to copy the shared library into a location your OS is aware of in order
to load the library dynamically at runtime. For example, on Ubuntu 20.04 this location is `/usr/lib`.

Having done that, you will now be able to build the application using the `make build` command line.

If you wish to build Docker images of these services, you can do so via the `make docker` command line. Prior to doing so, you will need to copy the `libiota_streams_c.so` library
referenced above to the following path: `internal/subscriber/streams/iota/include`. This will facilitate the copy of the library into the relevant Docker images.

# Makefile execution

If you build the services from source, you can run them locally. There are several different permutations for the supporting services
which are triggered by the Makefile. Unless otherwise specified all services use MQTT for pub/sub, ArangoDB for persisting the Alvarium
DCF graph and MongoDB for the example "business database." The Mongo database is populated with example business data by the example applications
provided in [Go](https://github.com/project-alvarium/example-go) and [Java](https://github.com/project-alvarium/example-java).

- `make run` will start the services locally with a small delay between each.
- `make run_docker` uses the scripts/docker/docker-compose.yml file to bring up all of the services and their supporting applications.
- `make_iota` will use the IOTA Tangle for pub/sub of DCF events rather than MQTT. This will require a functional instance of the [IOTA Streams Author](https://github.com/project-alvarium/streams-author)
- `make run_iota_opa` will use the IOTA Tangle for pub/sub of DCF events rather than MQTT. This will require a functional instance of the [IOTA Streams Author](https://github.com/project-alvarium/streams-author). 
As indicated in the `make` argument this option also supports OPA for applying annotation weights by policy when calculating a score. You should enable the OPA server first via the scripts/policies/Dockerfile.
- `make run_opa` executes the services using all defaults except for OPA policy enablement. See above for how to start the OPA server via Docker.

# Scoring overview

Please see the `DCF_Scoring_Method_Proposed.pdf` document in the root of this repo for a complete flow diagram of how these services interact.