package auth

import (
	"context"
	"roboserver/database"
	"roboserver/shared"
	"time"
)

// StartHeartbeatLoop resets the Redis TTL for a robot on each tick.
// It stops when ctx is cancelled or the done channel is closed.
func StartHeartbeatLoop(ctx context.Context, rds *database.RedisHandler, uuid string, done <-chan struct{}) {
	ttl := shared.AppConfig.Database.Redis.TTL()
	// Refresh at half the TTL interval to avoid races
	interval := ttl / 2
	if interval < time.Second {
		interval = time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			if err := rds.RefreshHeartbeat(ctx, uuid, ttl); err != nil {
				shared.DebugPrint("Heartbeat refresh failed for %s: %v", uuid, err)
			}
		}
	}
}
