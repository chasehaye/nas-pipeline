# Documentation

An index of every doc in the repo. Start with the
[root README](../README.md) for the project overview; everything else is below.

## Project & operations

| Doc | What it covers |
|---|---|
| [Getting started](getting-started.md) | Local setup, the `make` targets, dev/prod port maps |
| [Testing](testing.md) | Test strategy, the pyramid, the synthetic-input pattern |
| [Security](SECURITY.md) | Vulnerability reporting, CUI / credentials handling |
| [Roadmap](roadmap.md) | Known limitations and planned work |
| [CI/CD](../.github/README.md) | The deploy pipeline |

## Post-mortems

| Post-mortem | What it covers |
|---|---|
| [Corrupt batch (2026-09-03)](post-mortems/2026-09-03-corrupt-batch.md) | A corrupt Kafka record froze `normalizer`: cause, recovery, proposed fix |
| [Stalled consumer](post-mortems/stalled-consumer.md) | Silent consumer stalls have no alerting: diagnosis and proposed fix |

## Services

| Service | Role |
|---|---|
| [bridge](../bridge/README.md) | SWIM (JMS) to Kafka `fixm.raw` |
| [normalizer](../normalizer/README.md) | FIXM XML to per-flight JSON |
| [filter](../filter/README.md) | LADD compliance gate |
| [cache-writer](../cache-writer/README.md) | `fixm.filtered` to Redis (live state) |
| [database-writer](../database-writer/README.md) | `fixm.filtered` to Postgres/TimescaleDB (history) |
| [api](../api/README.md) | Read API over Redis (live) + Postgres (history) |
| [web](../web/README.md) | Live MapLibre flight map |
| [ladd-admin](../ladd-admin/README.md) | Secure LADD-list upload (control plane) |

Shared module: [platform](../platform/README.md), the observability and Kafka
plumbing every service imports.

<!--
TODO (author): pages still to write:
  - architecture.md   cross-cutting: Kafka topology, delivery guarantees,
                      the platform/ shared module, fail-closed/fail-fast.
  - data-model.md     FIXM -> normalized JSON -> Redis hash -> Postgres schema.
  - deployment.md     k3s single-node, Cloudflare tunnel, storage, secrets.
-->
