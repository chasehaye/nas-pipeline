-- 0001 initial schema: airports -> flights -> positions (+ TimescaleDB).
-- Non-destructive (IF NOT EXISTS) so it applies to a fresh DB or records
-- harmlessly on a DB that already has the schema.

CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE IF NOT EXISTS airports (
    icao_code text PRIMARY KEY,
    name      text,
    lat       double precision,
    lon       double precision
);

CREATE TABLE IF NOT EXISTS flights (
    gufi                   text PRIMARY KEY,
    call_sign              text,
    registration           text,
    aircraft_type          text,
    origin                 text REFERENCES airports (icao_code),
    destination            text REFERENCES airports (icao_code),
    status                 text,
    status_time            timestamptz,
    drop_count             integer NOT NULL DEFAULT 0,
    reactivation_count     integer NOT NULL DEFAULT 0,
    actual_departure_time  timestamptz,
    actual_arrival_time    timestamptz,
    first_seen             timestamptz NOT NULL DEFAULT now(),
    last_seen              timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS flights_origin_idx      ON flights (origin);
CREATE INDEX IF NOT EXISTS flights_destination_idx ON flights (destination);

CREATE TABLE IF NOT EXISTS positions (
    time      timestamptz      NOT NULL,
    gufi      text             NOT NULL REFERENCES flights (gufi),
    lat       double precision,
    lon       double precision,
    alt       double precision,
    heading   double precision,
    speed_kt  double precision,
    status    text
);

SELECT create_hypertable('positions', 'time', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS positions_gufi_time_idx ON positions (gufi, time DESC);

ALTER TABLE positions SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'gufi',
    timescaledb.compress_orderby   = 'time DESC'
);
SELECT add_compression_policy('positions', INTERVAL '7 days', if_not_exists => TRUE);
SELECT add_retention_policy('positions', INTERVAL '90 days', if_not_exists => TRUE);
