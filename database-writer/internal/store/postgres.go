package store

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chasehaye/nas-pipeline/database-writer/internal/flight"
	"github.com/chasehaye/nas-pipeline/database-writer/internal/migrate"
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

// Migrate applies any pending schema migrations.
func (s *Store) Migrate(ctx context.Context) error { return migrate.Run(ctx, s.pool) }

const upsertAirportSQL = `
INSERT INTO airports (icao_code, lat, lon)
VALUES ($1, $2, $3)
ON CONFLICT (icao_code) DO UPDATE SET
    lat = COALESCE(airports.lat, EXCLUDED.lat),
    lon = COALESCE(airports.lon, EXCLUDED.lon)`

// upsertFlightSQL keeps the first non-null value for identity fields
// (COALESCE(existing, new)), advances status only with a newer message
// (status_time guard), and counts drop episodes / reactivations on the exact
// status edges. last_seen tracks processing time for staleness queries.
const upsertFlightSQL = `
INSERT INTO flights (
    gufi, call_sign, registration, aircraft_type, origin, destination,
    status, status_time, actual_departure_time, actual_arrival_time
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (gufi) DO UPDATE SET
    call_sign             = COALESCE(flights.call_sign, EXCLUDED.call_sign),
    registration          = COALESCE(flights.registration, EXCLUDED.registration),
    aircraft_type         = COALESCE(flights.aircraft_type, EXCLUDED.aircraft_type),
    origin                = COALESCE(flights.origin, EXCLUDED.origin),
    destination           = COALESCE(flights.destination, EXCLUDED.destination),
    actual_departure_time = COALESCE(flights.actual_departure_time, EXCLUDED.actual_departure_time),
    actual_arrival_time   = COALESCE(flights.actual_arrival_time, EXCLUDED.actual_arrival_time),
    drop_count = flights.drop_count + CASE
        WHEN EXCLUDED.status_time >= flights.status_time
         AND flights.status IS DISTINCT FROM 'DROPPED'
         AND EXCLUDED.status = 'DROPPED' THEN 1 ELSE 0 END,
    reactivation_count = flights.reactivation_count + CASE
        WHEN EXCLUDED.status_time >= flights.status_time
         AND flights.status = 'DROPPED'
         AND EXCLUDED.status = 'ACTIVE' THEN 1 ELSE 0 END,
    status = CASE
        WHEN flights.status_time IS NULL OR EXCLUDED.status_time >= flights.status_time
        THEN EXCLUDED.status ELSE flights.status END,
    status_time = GREATEST(flights.status_time, EXCLUDED.status_time),
    last_seen = now()`

const insertPositionSQL = `
INSERT INTO positions (time, gufi, lat, lon, alt, heading, speed_kt, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

// RecordBatch writes a whole batch in one transaction, pipelined via pgx.Batch:
// all statements are sent in a single round-trip instead of one at a time, so
// throughput isn't bound by per-statement network latency. Queue order is
// airports -> flights -> positions (the FK chain), and flights stay in message
// order so the per-GUFI status_time counters remain correct.
func (s *Store) RecordBatch(ctx context.Context, flights []flight.Flight) error {
	if len(flights) == 0 {
		return nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	batch := &pgx.Batch{}

	// Airports first (FK parents), deduped across the batch — a batch of
	// hundreds of flights usually touches only a few dozen distinct airports.
	airports := make(map[string]flight.Coord, len(flights))
	for _, f := range flights {
		addAirport(airports, f.Origin, f.OriginCoord)
		addAirport(airports, f.Destination, f.DestinationCoord)
	}
	for code, c := range airports {
		batch.Queue(upsertAirportSQL, code, nfloat(c.Lat, c.OK), nfloat(c.Lon, c.OK))
	}

	// Flights, in message order.
	for _, f := range flights {
		stime := parseTimeOr(f.Timestamp, time.Now().UTC())
		batch.Queue(upsertFlightSQL,
			f.Gufi, nz(f.CallSign), nz(f.Registration), nz(f.AircraftType),
			nz(f.Origin), nz(f.Destination), nz(f.Status), stime,
			ntime(f.ActualDepartureTime), ntime(f.ActualArrivalTime),
		)
	}

	// Positions (FK children of flights).
	for _, f := range flights {
		if !f.HasPosition {
			continue
		}
		stime := parseTimeOr(f.Timestamp, time.Now().UTC())
		ptime := parseTimeOr(f.Position.Time, stime)
		batch.Queue(insertPositionSQL,
			ptime, f.Gufi, f.Position.Lat, f.Position.Lon,
			nfloat(f.Position.Alt, f.Position.HasAlt),
			nfloat(f.Position.Heading, f.Position.HasHeading),
			nfloat(f.Position.SpeedKt, f.Position.HasHeading),
			nz(f.Status),
		)
	}

	br := tx.SendBatch(ctx, batch)
	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			br.Close()
			return err
		}
	}
	if err := br.Close(); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// addAirport records a distinct airport code, preferring an entry that carries
// coordinates over one that doesn't.
func addAirport(m map[string]flight.Coord, code string, c flight.Coord) {
	if code == "" {
		return
	}
	if existing, ok := m[code]; ok {
		if !existing.OK && c.OK {
			m[code] = c
		}
		return
	}
	m[code] = c
}

func nz(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func nfloat(v float64, ok bool) any {
	if !ok {
		return nil
	}
	return v
}

func ntime(s string) any {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return nil
}

func parseTimeOr(s string, fb time.Time) time.Time {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return fb
}
