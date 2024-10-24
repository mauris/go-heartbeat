package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/mauris/go-heartbeat"
)

// main is an example code of how to run HeartbeatManager
func main() {
	// Create a new HeartbeatManager with Redis connection, lease TTL, and heartbeat interval
	leaseDuration := 10 * time.Second
	heartbeatInterval := 5 * time.Second
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Ping Redis to check connection
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("failed to connect to Redis: %v", err)
	}

	manager, err := heartbeat.NewHeartbeatManager(heartbeat.HeartbeatManagerConfig{
		RedisClient:       client,
		LeaseDuration:     leaseDuration,
		HeartbeatInterval: heartbeatInterval,
		Namespace:         "app",
	})
	if err != nil {
		log.Fatalf("Failed to initialize heartbeat manager: %v", err)
	}

	// Register instance in Redis
	err = manager.Register()
	if err != nil {
		log.Fatalf("Failed to register instance: %v", err)
	}

	// Start the heartbeat process
	manager.Start()

	// Simulate the instance running for 60 seconds
	fmt.Println("Instance running with ID:", manager.GetInstanceID())
	time.Sleep(60 * time.Second)

	// Stop the heartbeat when shutting down
	manager.Stop()

	fmt.Println("Instance shutting down")
}
