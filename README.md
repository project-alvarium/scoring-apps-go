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
