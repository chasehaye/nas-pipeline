package store

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/chasehaye/nas-pipeline/redis-service/internal/flight"
)

type Store struct {
	rdb    *redis.Client
	prefix string
	ttl    time.Duration
}

func New(addr, password string, db int, prefix string, ttl time.Duration) *Store {
	return &Store{
		rdb: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
		prefix: prefix,
		ttl:    ttl,
	}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.rdb.Ping(ctx).Err()
}

// UpsertFlight writes the current position for a GUFI as a single hash and
// re-arms its TTL. Overwriting (not appending) is the "single log per GUFI":
// the hash always holds the latest known position, and the TTL drops it from
// the active set once updates stop.
func (s *Store) UpsertFlight(ctx context.Context, f flight.Flight) error {
	key := s.prefix + f.Gufi
	fields := map[string]any{
		"gufi":         f.Gufi,
		"callSign":     f.CallSign,
		"registration": f.Registration,
		"status":       f.Status,
		"lat":          f.Position.Lat,
		"lon":          f.Position.Lon,
		"alt":          f.Position.AltValue,
		"altUom":       f.Position.AltUOM,
		"positionTime": f.Position.PositionTime,
		"timestamp":    f.Timestamp,
		"updatedAt":    time.Now().UTC().Format(time.RFC3339),
	}

	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, s.ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Store) Close() error { return s.rdb.Close() }
