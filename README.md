# Heartbeat Manager

A simple heartbeat manager for managing instance UUID registrations and lease renewals using Redis. This module ensures that each running instance has a globally unique identifier and automatically renews its lease at specified intervals.

## Features

- Register instances with a unique UUID.
- Automatic lease broadcasting and renewal via Redis.
- Collision detection for instance IDs.
- Customizable lease duration and heartbeat interval.

## Installation

To use this module, include it in your Go project:

```bash
go get github.com/mauris/go-heartbeat
```

## Usage

### Importing

Import the package in your Go code:

```go
// ... other imports
import "github.com/mauris/go-heartbeat"

func ServerInit(ctx context.Context) {
    // Create a Redis client
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    // Define your heartbeat manager configuration
    config := heartbeat.HeartbeatManagerConfig{
        RedisClient:       redisClient,
        LeaseDuration:     10 * time.Second,
        HeartbeatInterval: 5 * time.Second,
        Namespace:         "myapp",
    }

    // Create a new heartbeat manager
    manager, err := heartbeat.NewHeartbeatManager(config)
    if err != nil {
        panic(err)
    }

    // Register the instance
    if err := manager.Register(); err != nil {
        panic(err)
    }

    // Start the heartbeat
    manager.Start()

    // Stop the heartbeat when a shutdown is signaled
    go func() {
        for {
            select {
			case <-ctx.Done():
                manager.Stop()
				return
            }
        }
    }()
}
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.
