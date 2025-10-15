# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS builder

RUN apk add --no-cache build-base

WORKDIR /build

# Download Git submodules
# COPY .git ./.git
# RUN git -c submodule.ui.update=none submodule update --init --recursive

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

# Transfer source code
COPY *.go ./

# Build
RUN go build -o /dist/forwarder


# Test
FROM builder AS run-test-stage
# COPY i18n ./i18n
RUN go test -v ./...

FROM scratch AS build-release-stage

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /dist /app

WORKDIR /app

ENTRYPOINT ["/app/forwarder"]