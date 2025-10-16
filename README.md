# Forwarder
Telegram forwarder bot.

## Instructions

First of all, you should set up some environment variables:
```
cp .env.example .env
nano .env
```

## Usage

### Run locally with Makefile
To run with the required CGO_LDFLAGS:
```
make run
```
To build the binary:
```
make build
```

### Run with Docker
Build the image:
```
make docker
```
Run interactively (required for PIN/password input):
```
docker run -it ghcr.io/birabittoh/forwarder:main
```
Or use docker-compose:
```
docker-compose -f compose.yaml up
```
The compose file mounts `tdlib-db` and `tdlib-files` directories for persistent storage.

### PIN and Telegram Password
When starting the bot, you will be prompted to enter your Telegram PIN and password interactively in the terminal. This is required for authentication.

If you need non-interactive authentication, you must modify `main.go` to read PIN/password from environment variables or files.

## Test and debug locally
```
go test -v ./...
make run
```
