package heartbeat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

const (
	redisInstanceIDKeyFormat = "%s:go-heartbeat:instance:%s"
)

type HeartbeatManager interface {
	Register() error
	Start()
	Stop()
	GetInstanceID() string
}

type heartbeatManagerImpl struct {
	client            *redis.Client
	instanceID        string
	isRegistered      bool
	leaseDuration     time.Duration
	heartbeatInterval time.Duration
	namespace         string
	ctx               context.Context
	cancelFunc        context.CancelFunc
}

type HeartbeatManagerConfig struct {
	RedisClient       *redis.Client
	LeaseDuration     time.Duration
	HeartbeatInterval time.Duration
	Namespace         string
}

var (
	ErrInstanceIDCollision = errors.New("instance id collision")
	ErrLeaseExpired        = errors.New("lease expired")
)

var (
	getUUIDString = uuid.New().String
	timeNewTicker = time.NewTicker
)

// NewHeartbeatManager creates a new heartbeat manager
func NewHeartbeatManager(config HeartbeatManagerConfig) (HeartbeatManager, error) {
	if config.LeaseDuration < config.HeartbeatInterval {
		return nil, fmt.Errorf("lease duration %v less than heartbeat interval %v", config.LeaseDuration, config.HeartbeatInterval)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	return &heartbeatManagerImpl{
		client:            config.RedisClient,
		leaseDuration:     config.LeaseDuration,
		heartbeatInterval: config.HeartbeatInterval,
		isRegistered:      false,
		namespace:         config.Namespace,
		ctx:               ctx,
		cancelFunc:        cancelFunc,
	}, nil
}

// Register creates a new UUID for the instance and registers it to Redis with a lease duration
func (h *heartbeatManagerImpl) Register() (err error) {
	h.instanceID = getUUIDString()
	success, err := h.client.SetNX(h.ctx, fmt.Sprintf(redisInstanceIDKeyFormat, h.namespace, h.instanceID), h.instanceID, h.leaseDuration).Result()
	if err != nil {
		return
	}

	if !success {
		return ErrInstanceIDCollision
	}
	h.isRegistered = true
	return
}

// renewLease renews the lease by extending the lease expiry.
func (h *heartbeatManagerImpl) renewLease() (err error) {
	ttl, err := h.client.TTL(h.ctx, fmt.Sprintf(redisInstanceIDKeyFormat, h.namespace, h.instanceID)).Result()
	if err != nil {
		return
	}

	if ttl > 0 {
		// Renew the lease by resetting the expiry
		err = h.client.Expire(h.ctx, fmt.Sprintf(redisInstanceIDKeyFormat, h.namespace, h.instanceID), h.leaseDuration).Err()
		if err != nil {
			return
		}
	} else {
		err = ErrLeaseExpired
	}
	return
}

// Start starts the heartbeat routine to renew the lease at regular intervals.
func (h *heartbeatManagerImpl) Start() {
	ticker := timeNewTicker(h.heartbeatInterval)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				err := h.renewLease()
				if err != nil {
					return
				}
			case <-h.ctx.Done():
				return
			}
		}
	}()
}

// Stop stops the heartbeat routine.
func (hb *heartbeatManagerImpl) Stop() {
	hb.cancelFunc()
}

// GetInstanceID returns the instance UUID.
func (h *heartbeatManagerImpl) GetInstanceID() string {
	return h.instanceID
}
