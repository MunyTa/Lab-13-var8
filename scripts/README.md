# Auto-scaling Agent

This service automatically scales HR agents based on queue length.

## Features

- Monitors NATS queue length via `/varz` endpoint
- Automatically scales agents when queue exceeds threshold
- Uses Docker API to create/remove agent instances
- Logs all scaling events

## Configuration

Environment variables:
- `QUEUE_THRESHOLD` - queue length threshold to trigger scaling (default: 10)
- `MAX_REPLICAS` - maximum number of replicas per agent (default: 5)
- `CHECK_INTERVAL` - interval between checks in seconds (default: 5)

## Usage

```bash
go run scripts/auto_scaler.go
```

## Integration

The auto-scaler integrates with:
- NATS monitoring API for queue stats
- Docker API for container management
- Redis for state (via orchestrator)

## Scaling Logic

1. Poll NATS `/varz` endpoint for subscription count
2. If count > QUEUE_THRESHOLD, scale up agents
3. Create new container instances up to MAX_REPLICAS
4. NATS queue groups distribute load automatically
