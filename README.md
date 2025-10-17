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
make
```

### Run with Docker
Build the image:
```
make docker
```
Run interactively (required for PIN/password input):
```
docker run -it --rm \
  --name forwarder \
  --env-file .env \
  -v "$(pwd)/comment.md:/app/comment.md:ro" \
  -v "$(pwd)/tdlib-db:/app/tdlib-db" \
  -v "$(pwd)/tdlib-files:/app/tdlib-files" \
  --log-driver json-file \
  --log-opt max-size=1m \
  ghcr.io/birabittoh/forwarder:main
```
Or use docker-compose:
```
docker-compose -f compose.yaml up
```
The compose file mounts `tdlib-db` and `tdlib-files` directories for persistent storage.

### PIN and Telegram Password
When starting the bot, you will be prompted to enter your Telegram PIN and password interactively in the terminal. This is required for authentication.

## Test and debug locally
```
go test -v ./...
make run
```

## License
Forwarder is provided under the MIT license.