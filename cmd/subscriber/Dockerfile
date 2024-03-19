# Docker image for subscriber-go DCF scoring micro service

ARG BUILDER_BASE=ubuntu:20.04
FROM ${BUILDER_BASE} AS builder

WORKDIR /dcf

#RUN apt update && apt install make git openssl curl bash
RUN apt-get update -y -q && apt-get upgrade -y -q
#Install additional tooling
RUN DEBIAN_FRONTEND=noninteractive apt-get install --no-install-recommends -y -q git openssl curl build-essential ca-certificates
#Install Go
RUN curl -s https://storage.googleapis.com/golang/go1.22.0.linux-amd64.tar.gz| tar -v -C /usr/local -xz

#Set PATH to include Go binaries
ENV PATH="${PATH}:/usr/local/go/bin"

RUN echo $PATH

COPY go.mod .

RUN go mod download

COPY . .
RUN make cmd/subscriber/subscriber-go

#Next image - Copy built Go binary into new workspace
FROM ${BUILDER_BASE}

WORKDIR /
COPY --from=builder /dcf/cmd/subscriber/subscriber-go /
COPY --from=builder /dcf/cmd/subscriber/res/docker/config.json /res/config.json
COPY --from=builder /dcf/cmd/subscriber/res/docker/config-mqtt.json /res/config-mqtt.json

ENTRYPOINT ["/subscriber-go", "-cfg=./res/config-mqtt.json"]