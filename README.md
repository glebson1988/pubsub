# Peril Pub/Sub

A small multiplayer CLI game used to learn pub/sub concepts with RabbitMQ. Players move units, trigger wars, and generate game logs. The server consumes game logs and writes them to `game.log`. The project exercises topic routing, durable queues, gob encoding, and backpressure tuning (prefetch).

## Features
- Topic exchange for game events and logs.
- Gob-encoded `GameLog` publishing and consumption.
- Durable `game_logs` queue and disk logging.
- Spam command to generate high log volume for backpressure tests.
- Prefetch limit (QoS) to keep consumers fair.

## Requirements
- Go 1.20+ (or compatible).
- RabbitMQ running locally on `amqp://guest:guest@localhost:5672/`.

## Quickstart
Start RabbitMQ (if you do not already have it running):
```bash
./rabbit.sh
```

Start the server:
```bash
go run ./cmd/server
```

Start one or more clients:
```bash
go run ./cmd/client
```

## Client Commands
```text
spawn <location> <rank>
move <location> <unitID> <unitID> ...
status
spam <n>
help
quit
```

## How Logging Works
- Clients publish `routing.GameLog` messages to `peril_topic` with routing key `game_logs.<username>`.
- The server subscribes to `game_logs.*` and writes logs to `game.log`.

## Backpressure Demo
Generate a lot of logs:
```bash
spam 1000
```

Run many log consumers:
```bash
./multiserver.sh 100
```

Prefetch is capped at 10 per consumer to prevent a single server from monopolizing messages.

## Project Structure
```text
cmd/client     CLI client
cmd/server     Log consumer + pause/resume broadcaster
internal/pubsub Pub/Sub helpers (PublishJSON/Gob, SubscribeJSON/Gob)
internal/gamelogic Game state and war logic
internal/routing   Exchange keys and models
```

## Notes
- Logs are written to `game.log` in the repo root.
- For best results, run two clients to trigger wars and produce logs.
