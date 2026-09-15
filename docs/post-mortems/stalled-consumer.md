# Post-mortem: a stalled consumer went unnoticed (observability gap)

## Summary

During the 2026-09-03 corruption incident, `normalizer` stopped consuming and the
whole pipeline stalled, but nothing alerted. The stall was found by hand and
diagnosed manually. The problem written up here is not the specific cause (that is
[the corrupt-batch post-mortem](2026-09-03-corrupt-batch.md)) but that a stalled
consumer produces **no alert**, even though the signal to detect it already exists.

## The problem

A pipeline stage can stop making progress while everything looks healthy:

- pods stay `Running` (the process is alive, just wedged or erroring in a loop),
- the upstream stage keeps producing, so data keeps arriving,
- consumer lag climbs, but nothing watches it.

The result is a silent outage: throughput is zero, and the only way it gets noticed
is a human spotting that output stopped.

## How to diagnose it (until alerting exists)

The path that found the 2026-09-03 stall, generalized:

1. **Confirm which stage is stuck.** Check pods, and check the upstream stage is
   still producing (`kubectl logs`). If upstream is healthy and producing, the
   problem is the consumer downstream of it.
2. **Read the stuck consumer's logs.** Only startup lines and no errors is itself a
   signal (wedged before doing work). Grep for the failure paths:
   `publish failed|commit failed|consume error|dlq|progress|panic`.
3. **Check the consumer-group offset** (run twice, ~10s apart). Kafka's internal
   listener is `kafka:29092`:
   ```bash
   kubectl exec -n nas kafka-0 -- /opt/kafka/bin/kafka-consumer-groups.sh \
     --bootstrap-server kafka:29092 --describe --group <group>
   ```
   - CURRENT-OFFSET **frozen** + LAG climbing = wedged.
   - Group not found / no members = never connected (broker address / networking).
   - Offset advancing = it is working; likely just a backlog draining.
4. **Narrow with metrics** on the ops port (`2112`):
   `envelopes_processed_total` at 0 means it never got a message out of the consumer
   (stuck in `Fetch`, usually a corrupt record); rising `publish_errors_total` means
   the producer side.

## Proposed solution

- **Alert on the stall signature**: a Prometheus rule for `lag > 0` AND
  current-offset flat for N minutes. The lag is already scraped by `kafka-exporter`,
  so this is a few lines of PromQL and would turn a manual hunt into an instant
  page. → [roadmap](../roadmap.md)
- **Corrupt-batch auto-skip** removes the most common wedged-in-`Fetch` cause.
  → [corrupt-batch post-mortem](2026-09-03-corrupt-batch.md)

## Lessons

- The stall was invisible for lack of alerting, not lack of signal.
- "Pods are `Running`" and "no error logs" are both fully compatible with a total
  outage.
