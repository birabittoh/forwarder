# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS builder
WORKDIR /build

# Install build dependencies
RUN apk add --no-cache \
    build-base \
    cmake \
    openssl-dev \
    zlib-dev \
    linux-headers \
    gperf

# Copy and build tdlib from source
COPY tdlib/ /tdlib/
RUN mkdir -p /tdlib/build
RUN rm -rf /tdlib/build/*
WORKDIR /tdlib/build
RUN cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/usr/local ..
RUN cmake --build . -j$(nproc)
RUN cmake --install .

# Set up Go build environment
WORKDIR /build
ENV CGO_ENABLED=1
ENV CGO_LDFLAGS="-ltdjson"

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

# Transfer source code and build
COPY *.go ./
COPY config/ ./config/
COPY forwarder/ ./forwarder/
RUN go build -o /dist/forwarder

# Test stage
FROM builder AS run-test-stage
RUN go test -v ./...

# Final stage
FROM alpine:3 AS build-release-stage
RUN apk add --no-cache libstdc++
COPY --from=builder /usr/local/lib/libtd*.so* /usr/local/lib/
COPY --from=builder /dist/forwarder /app/forwarder
WORKDIR /app
ENTRYPOINT ["/app/forwarder"]
