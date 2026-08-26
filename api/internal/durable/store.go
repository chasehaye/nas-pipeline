package durable

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }
func (s *Store) Close()                         { s.pool.Close() }

// Flight is a flight's stored metadata. Nullable columns are pointers so they
// serialize as absent rather than zero values.
type Flight struct {
	Gufi              string     `json:"gufi"`
	CallSign          *string    `json:"callSign,omitempty"`
	Registration      *string    `json:"registration,omitempty"`
	AircraftType      *string    `json:"aircraftType,omitempty"`
	Origin            *string    `json:"origin,omitempty"`
	Destination       *string    `json:"destination,omitempty"`
	Status            *string    `json:"status,omitempty"`
	ActualDeparture   *time.Time `json:"actualDepartureTime,omitempty"`
	ActualArrival     *time.Time `json:"actualArrivalTime,omitempty"`
	DropCount         int        `json:"dropCount"`
	ReactivationCount int        `json:"reactivationCount"`
	FirstSeen         time.Time  `json:"firstSeen"`
	LastSeen          time.Time  `json:"lastSeen"`
}

// Position is one observation on a flight's track.
type Position struct {
	Time    time.Time `json:"time"`
	Lat     *float64  `json:"lat,omitempty"`
	Lon     *float64  `json:"lon,omitempty"`
	Alt     *float64  `json:"alt,omitempty"`
	Heading *float64  `json:"heading,omitempty"`
	SpeedKt *float64  `json:"speedKt,omitempty"`
	Status  *string   `json:"status,omitempty"`
}

// Record is a flight's metadata plus its full position track.
type Record struct {
	Flight Flight     `json:"flight"`
	Track  []Position `json:"track"`
}

const flightSQL = `
SELECT gufi, call_sign, registration, aircraft_type, origin, destination,
       status, actual_departure_time, actual_arrival_time,
       drop_count, reactivation_count, first_seen, last_seen
FROM flights WHERE gufi = $1`

const trackSQL = `
SELECT time, lat, lon, alt, heading, speed_kt, status
FROM positions WHERE gufi = $1 ORDER BY time`

// GetRecord returns a flight's metadata plus its ordered track, or (nil, nil)
// if the GUFI is unknown.
func (s *Store) GetRecord(ctx context.Context, gufi string) (*Record, error) {
	var f Flight
	err := s.pool.QueryRow(ctx, flightSQL, gufi).Scan(
		&f.Gufi, &f.CallSign, &f.Registration, &f.AircraftType, &f.Origin, &f.Destination,
		&f.Status, &f.ActualDeparture, &f.ActualArrival, &f.DropCount, &f.ReactivationCount,
		&f.FirstSeen, &f.LastSeen,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	rows, err := s.pool.Query(ctx, trackSQL, gufi)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	track := make([]Position, 0)
	for rows.Next() {
		var p Position
		if err := rows.Scan(&p.Time, &p.Lat, &p.Lon, &p.Alt, &p.Heading, &p.SpeedKt, &p.Status); err != nil {
			return nil, err
		}
		track = append(track, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &Record{Flight: f, Track: track}, nil
}
