package store

import (
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type Store struct {
	rdb    *redis.Client
	prefix string
}

func New(addr, password string, db int, prefix string) *Store {
	return &Store{
		rdb: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
		prefix: prefix,
	}
}

func (s *Store) Ping(ctx context.Context) error { return s.rdb.Ping(ctx).Err() }
func (s *Store) Close() error                    { return s.rdb.Close() }


type Flight struct {
	Gufi         string  `json:"gufi"`
	CallSign     string  `json:"callSign"`
	Registration string  `json:"registration,omitempty"`
	Status       string  `json:"status"`
	Lat          float64 `json:"lat"`
	Lon          float64 `json:"lon"`
	Alt          float64 `json:"alt,omitempty"`
	AltUom       string  `json:"altUom,omitempty"`
	Heading      float64 `json:"heading,omitempty"`
	SpeedKt      float64 `json:"speedKt,omitempty"`
	PositionTime string  `json:"positionTime,omitempty"`
	Timestamp    string  `json:"timestamp,omitempty"`
	UpdatedAt    string  `json:"updatedAt,omitempty"`
}


func (s *Store) ListFlights(ctx context.Context) ([]Flight, error) {
	var keys []string
	iter := s.rdb.Scan(ctx, 0, s.prefix+"*", 500).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	flights := make([]Flight, 0, len(keys))
	if len(keys) == 0 {
		return flights, nil
	}

	pipe := s.rdb.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(keys))
	for i, k := range keys {
		cmds[i] = pipe.HGetAll(ctx, k)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}
	for _, cmd := range cmds {
		m, err := cmd.Result()
		if err != nil || len(m) == 0 {
			continue
		}
		flights = append(flights, flightFromHash(m))
	}
	return flights, nil
}

func (s *Store) GetFlight(ctx context.Context, gufi string) (Flight, bool, error) {
	m, err := s.rdb.HGetAll(ctx, s.prefix+gufi).Result()
	if err != nil {
		return Flight{}, false, err
	}
	if len(m) == 0 {
		return Flight{}, false, nil
	}
	return flightFromHash(m), true, nil
}

func flightFromHash(m map[string]string) Flight {
	return Flight{
		Gufi:         m["gufi"],
		CallSign:     m["callSign"],
		Registration: m["registration"],
		Status:       m["status"],
		Lat:          parseF(m["lat"]),
		Lon:          parseF(m["lon"]),
		Alt:          parseF(m["alt"]),
		AltUom:       m["altUom"],
		Heading:      parseF(m["heading"]),
		SpeedKt:      parseF(m["speedKt"]),
		PositionTime: m["positionTime"],
		Timestamp:    m["timestamp"],
		UpdatedAt:    m["updatedAt"],
	}
}

func parseF(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
