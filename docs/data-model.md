# Data model

How a FIXM flight becomes the compact per-flight JSON the pipeline carries on
`fixm.normalized` (and, unchanged, `fixm.filtered`). The `normalizer` parses the
raw FIXM XML and keeps only the fields anything downstream actually uses.

```
raw FIXM XML (fixm.raw)  ->  normalizer  ->  per-flight JSON (fixm.normalized)
   dozens of subtrees          keep-set        the fields below, nothing else
```

## The keep-set

These are the only fields the normalizer emits, because they are the only ones
read by a downstream service (`filter`, `cache-writer`, `database-writer`; the
`api` only reads what the writers stored, so it adds nothing).

| JSON path (under `flight`) | Source (FIXM) | Consumed by |
|---|---|---|
| `timestamp` | `@timestamp` | cache, db |
| `gufi.code` | `gufi` text | filter, cache, db (the per-flight key) |
| `flightIdentification.aircraftIdentification` | `@aircraftIdentification` | filter, cache, db (call sign) |
| `flightStatus.fdpsFlightStatus` | `@fdpsFlightStatus` | cache, db (lifecycle) |
| `aircraftDescription.registration` | `@registration` | filter, cache, db |
| `aircraftDescription.aircraftType.icaoModelIdentifier` | element | db |
| `departure.departurePoint` | `@departurePoint` | db (origin) |
| `departure.runwayPositionAndTime.runwayTime.actual.time` | `@time` | db (actual departure) |
| `departure.departureAerodrome.point.location.pos` | element | db (origin coords, rare) |
| `arrival.arrivalPoint` | `@arrivalPoint` | db (destination) |
| `arrival.runwayPositionAndTime.runwayTime.actual.time` | `@time` | db (actual arrival) |
| `arrival.arrivalAerodrome.point.location.pos` | element | db (dest coords, rare) |
| `enRoute.positions[].positionTime` | `@positionTime` | cache, db |
| `enRoute.positions[].altitude.value` / `.uom` | text / `@uom` | cache, db |
| `enRoute.positions[].location.pos` | `position/position/location/pos` | cache, db (lat lon) |
| `enRoute.positions[].trackVelocity.x.value` / `.y.value` | text | cache, db (heading + speed) |

Two non-obvious points baked into this list:

- **Heading and speed come from `trackVelocity` (x, y), not `actualSpeed`.** Both
  writers derive heading and ground speed from the velocity vector
  (`atan2`/`hypot`), so the `actualSpeed`/`surveillance` field, despite looking
  like "the speed", is never read and is dropped.
- **Only the latest position is used.** The writers loop `positions[]` and keep
  the one with the max `positionTime`. The slice is still emitted; the pick is a
  downstream concern.

## What is dropped (and why)

Everything else in the FIXM `NasFlightType` is intentionally not mapped: the
whole `agreed/route` subtree (expandedRoute, estimatedElapsedTime, adapted
routes, holdFix, segment), `coordination`, `operator`, `supplementalData`,
`assignedAltitude`/`requestedAltitude`/`interimAltitude`/`requestedAirspeed`,
`aircraftDescription` capabilities/accuracy, `flightPlan`, `controllingUnit`,
`originator`, `specialHandling`, boundary crossings / pointout / cleared /
beacon codes, target altitude/position, alternate aerodromes, estimated and
controlled runway times, and the sparse identity extras (`computerId`,
`siteSpecificPlanId`, `codeSpace`, `airborneHold`).

A few constants are dropped because they carry no information: `@system` is
always `ATL`, and the position `@uom` fields are always `FEET`/`KNOTS` across
every record observed.

## Why lean, not "capture everything"

A `dev_census` run over ~10M flights shows the core identity/position fields are
stable, while the long tail keeps sprouting new rarely-used fields (recent
additions: `agreed/route/segment`, `assignedAltitude/vfrOnTop`, a `point` on
`arrivalAerodromeAlternate`). Chasing full coverage is a moving target; the
keep-set is a fixed one.

This costs nothing in robustness: Go's `encoding/xml` **silently ignores any
element not mapped in the structs**. New or unused FIXM fields never cause an
error, they are simply not captured. The model only grows when we deliberately
add a field because something downstream will read it.

## Adding a field later

If a consumer needs a dropped field, add it in three places: the struct in
`normalizer/internal/fixm/models.go` (with the right `xml`/`json` tags), the
consumer's own parse struct, and a line in the table above. Candidates most
likely to earn their place: `aircraftDescription.aircraftAddress` (ICAO 24-bit
hex, for ADS-B joins) and `@centre` (ARTCC, a natural dashboard dimension).
