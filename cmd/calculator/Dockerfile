# Docker image for calculator-go DCF scoring micro service

ARG BUILDER_BASE=golang:1.22-alpine
FROM ${BUILDER_BASE} AS builder

WORKDIR /dcf

# Install additional tooling for build
RUN apk add build-base

#RUN apt update && apt install make git
RUN sed -e 's/dl-cdn[.]alpinelinux.org/nl.alpinelinux.org/g' -i~ /etc/apk/repositories
RUN apk update && apk add make git

COPY go.mod .

RUN go mod download

COPY . .
RUN make cmd/calculator/calculator-go

#Next image - Copy built Go binary into new workspace
FROM ${BUILDER_BASE}

WORKDIR /
COPY --from=builder /dcf/cmd/calculator/calculator-go /
COPY --from=builder /dcf/cmd/calculator/res/docker/config-mqtt.json /res/config-mqtt.json
COPY --from=builder /dcf/cmd/calculator/res/docker/config-mqtt-opa.json /res/config-mqtt-opa.json

ENTRYPOINT ["/calculator-go", "-cfg=./res/config-mqtt-opa.json", "-mode=production"]