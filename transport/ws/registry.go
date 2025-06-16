package ws

import (
	"fmt"

	"github.com/gomodule/redigo/redis"
)

// PresenceRegistry manages user presence information in Redis.
type PresenceRegistry struct {
	Prefix string      // Prefix for Redis keys
	Redis  *redis.Pool // Redis connection pool
}

// NewPresenceRegistry creates a new PresenceRegistry with the given Redis pool.
func NewPresenceRegistry(rds *redis.Pool) *PresenceRegistry {
	return &PresenceRegistry{
		Prefix: "ws:user:",
		Redis:  rds,
	}
}

// Add registers a podID for a userID in Redis, indicating the user is present on that pod.
func (r *PresenceRegistry) Add(userID, podID string) error {
	conn := r.Redis.Get()
	defer conn.Close()

	_, err := conn.Do("SADD", r.key(userID), podID)
	return err
}

// Remove unregisters a podID for a userID in Redis.
// If the user has no more pods, the Redis key for the user is deleted as a side effect.
func (r *PresenceRegistry) Remove(userID, podID string) error {
	conn := r.Redis.Get()
	defer conn.Close()

	key := r.key(userID)

	_, err := conn.Do("SREM", key, podID)
	if err != nil {
		return err
	}

	count, err := redis.Int(conn.Do("SCARD", key))
	if err != nil {
		return err
	}
	if count == 0 {
		_, _ = conn.Do("DEL", key)
	}
	return nil
}

// GetPods retrieves all podIDs where the userID is present.
func (r *PresenceRegistry) GetPods(userID string) ([]string, error) {
	conn := r.Redis.Get()
	defer conn.Close()

	return redis.Strings(conn.Do("SMEMBERS", r.key(userID)))
}

// Clear removes all presence data for a userID.
func (r *PresenceRegistry) Clear(userID string) error {
	conn := r.Redis.Get()
	defer conn.Close()

	_, err := conn.Do("DEL", r.key(userID))
	return err
}

// key generates the Redis key for a given userID.
func (r *PresenceRegistry) key(userID string) string {
	return fmt.Sprintf("%s%s", r.Prefix, userID)
}
