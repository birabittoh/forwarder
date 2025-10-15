# Forwarder
Telegram forwarder bot.

## Instructions

First of all, you should set up some environment variables:
```
cp .env.example .env
nano .env
```

### Run with Docker
Just run:
```
docker-compose -f docker-compose.yaml up -d
```

## Test and debug locally
```
go test -v ./...
go run .
```