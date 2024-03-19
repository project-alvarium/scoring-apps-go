# Docker image for populator-api-go DCF scoring micro service

ARG BUILDER_BASE=golang:1.22-alpine
FROM ${BUILDER_BASE} AS builder

WORKDIR /dcf

#RUN apt update && apt install make git
RUN sed -e 's/dl-cdn[.]alpinelinux.org/nl.alpinelinux.org/g' -i~ /etc/apk/repositories
RUN apk update && apk add make git

COPY go.mod .

RUN go mod download

COPY . .
RUN make cmd/populator-api/populator-api-go

#Next image - Copy built Go binary into new workspace
FROM ${BUILDER_BASE}

WORKDIR /
COPY --from=builder /dcf/cmd/populator-api/populator-api-go /
COPY --from=builder /dcf/cmd/populator-api/res/docker/config.json /res/config.json

ENTRYPOINT ["/populator-api-go", "-cfg=./res/config.json"]