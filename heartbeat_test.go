package heartbeat

import (
	"testing"
	"time"

	"github.com/go-redis/redismock/v8"
)

func TestHeartbeatManager_Register(t *testing.T) {
	defer func(currentRef *func() string) {
		getUUIDString = *currentRef
	}(&getUUIDString)
	getUUIDString = func() string {
		return "<uuid>"
	}

	// Create a Redis mock
	db, mock := redismock.NewClientMock()
	leaseDuration := 1 * time.Second
	heartbeatInterval := 100 * time.Millisecond
	namespace := "test"

	// Create the HeartbeatManager
	manager, err := NewHeartbeatManager(HeartbeatManagerConfig{
		RedisClient:       db,
		LeaseDuration:     leaseDuration,
		HeartbeatInterval: heartbeatInterval,
		Namespace:         namespace,
	})
	if err != nil {
		t.Fatalf("Failed to initialize heartbeat manager: %v", err)
	}

	// Mock the Redis SetNX command for registration
	mock.ExpectSetNX("test:go-heartbeat:instance:<uuid>", "<uuid>", leaseDuration).SetVal(true)

	// Test registration
	err = manager.Register()
	if err != nil {
		t.Fatalf("Failed to register instance: %v", err)
	}

	// Ensure the instance ID is set
	if manager.GetInstanceID() == "" {
		t.Fatal("Expected non-empty instance ID after registration")
	}

	// Test instance ID collision
	mock.ExpectSetNX("test:go-heartbeat:instance:"+manager.GetInstanceID(), manager.GetInstanceID(), leaseDuration).SetVal(false)

	err = manager.Register()
	if err != ErrInstanceIDCollision {
		t.Fatalf("Expected ErrInstanceIDCollision, got: %v", err)
	}

	// Ensure all expected commands were called
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("there were unfulfilled expectations: %s", err)
	}
}

func TestHeartbeatManager_RenewLease(t *testing.T) {
	defer func(currentRef *func() string) {
		getUUIDString = *currentRef
	}(&getUUIDString)
	getUUIDString = func() string {
		return "<uuid>"
	}
	defer func(currentRef *func(d time.Duration) *time.Ticker) {
		timeNewTicker = *currentRef
	}(&timeNewTicker)
	timeCh := make(chan time.Time)
	timeNewTicker = func(d time.Duration) *time.Ticker {
		return &time.Ticker{C: timeCh}
	}

	// Create a Redis mock
	db, mock := redismock.NewClientMock()
	leaseDuration := 1 * time.Second
	heartbeatInterval := 100 * time.Millisecond
	namespace := "test"

	// Create the HeartbeatManager
	manager, err := NewHeartbeatManager(HeartbeatManagerConfig{
		RedisClient:       db,
		LeaseDuration:     leaseDuration,
		HeartbeatInterval: heartbeatInterval,
		Namespace:         namespace,
	})
	if err != nil {
		t.Fatalf("Failed to initialize heartbeat manager: %v", err)
	}

	// Register the instance
	mock.ExpectSetNX("test:go-heartbeat:instance:<uuid>", "<uuid>", leaseDuration).SetVal(true)
	err = manager.Register()
	if err != nil {
		t.Fatalf("Failed to register instance: %v", err)
	}

	manager.Start()

	// Mock the TTL command for renewing the lease
	mock.ExpectTTL("test:go-heartbeat:instance:<uuid>").SetVal(leaseDuration)
	mock.ExpectExpire("test:go-heartbeat:instance:<uuid>", leaseDuration).SetVal(true)

	// Test renewing the lease
	timeCh <- time.Now()

	// Ensure all expected commands were called
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("there were unfulfilled expectations: %s", err)
	}
}

func TestHeartbeatManager_StartAndStop(t *testing.T) {
	defer func(currentRef *func() string) {
		getUUIDString = *currentRef
	}(&getUUIDString)
	getUUIDString = func() string {
		return "<uuid>"
	}

	// Create a Redis mock
	db, mock := redismock.NewClientMock()
	leaseDuration := 1 * time.Second
	heartbeatInterval := 100 * time.Millisecond
	namespace := "test"

	// Create the HeartbeatManager
	manager, err := NewHeartbeatManager(HeartbeatManagerConfig{
		RedisClient:       db,
		LeaseDuration:     leaseDuration,
		HeartbeatInterval: heartbeatInterval,
		Namespace:         namespace,
	})
	if err != nil {
		t.Fatalf("Failed to initialize heartbeat manager: %v", err)
	}

	// Mock the Redis SetNX command for registration
	mock.ExpectSetNX("test:go-heartbeat:instance:<uuid>", "<uuid>", leaseDuration).SetVal(true)

	// Register the instance
	err = manager.Register()
	if err != nil {
		t.Fatalf("Failed to register instance: %v", err)
	}

	// Start the heartbeat
	manager.Start()

	mock.ExpectTTL("test:go-heartbeat:instance:<uuid>").SetVal(leaseDuration)
	mock.ExpectExpire("test:go-heartbeat:instance:<uuid>", leaseDuration).SetVal(true)

	// Allow some time for the heartbeat to run
	time.Sleep(heartbeatInterval * 2)

	// Stop the heartbeat
	manager.Stop()

	// Ensure the instance ID is still accessible
	if manager.GetInstanceID() == "" {
		t.Fatal("Expected non-empty instance ID after stopping")
	}
}
