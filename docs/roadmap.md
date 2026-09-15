# Roadmap & known limitations

Honest limitations of the current build and the work intended next. A personal
project in its MVP phase (one prod cluster, live and ingesting), so this is frank.

## Next up (priority order)

1. **Corrupt-batch auto-skip** in the Go consumers. One bad record silently froze
   `normalizer` (2026-09-03) and only a manual offset-skip recovered it; make the
   consumers skip and dead-letter a corrupt batch instead of hanging.
   → [post-mortem](post-mortems/2026-09-03-corrupt-batch.md)
2. **Alert on a stalled consumer** (`lag > 0` and current-offset flat). A few lines
   of PromQL that would have caught that freeze in seconds instead of by hand.
3. **Local dev without real secrets.** A bundled non-CUI fake LADD list (so
   `filter` will start) plus a synthetic `fixm.raw` injector (so the pipeline runs
   with no live SWIM). → [why](getting-started.md#running-without-live-swim-local-data)
4. **Integration + end-to-end tests.** Per-service tests against the `make up`
   infra, and a root harness that feeds a synthetic message through to `/flights`.
   → [testing](testing.md)

## Backlog

**Durability & infrastructure**
- Kafka replication-factor ≥ 3 across separate hosts (`needs-hardware`): RF=1 can't self-heal a bad copy.
- Move monitoring (Prometheus / Grafana / kafka-exporter) in-cluster; it's Compose-only today.

**Testing**
- `web` component tests (Vitest).
- k3d dev-rehearsal overlay to test the Kubernetes deploy without real credentials.

**Features & data**
- Airport lat/lon backfill: `airports.lat/lon` are mostly `NULL` (the feed carries ICAO codes reliably, coordinates rarely).
- WebSockets instead of 4s polling, only if the live map ever needs sub-second latency.

**Security**
- Zero-downtime operator-key rotation in `ladd-admin` (accept two operator keys at once).

---

When you hit a limitation or defer a fix, add it here with a one-line note and a
link, so the reasoning isn't lost.
