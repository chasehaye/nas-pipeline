# Post-mortem: normalizer froze on a corrupt fixm.raw record (2026-09-03)

## Summary

A single corrupt record on the `fixm.raw` topic silently wedged the `normalizer`
consumer. Throughput from `bridge` onward dropped to zero and stayed there until a
manual offset-skip recovered it. Root cause was on-disk bit corruption (a CRC
mismatch), most likely from unstable host RAM; RF=1 meant there was no healthy
replica to recover from.

## Impact

- `normalizer` stopped processing entirely; nothing flowed to `fixm.normalized` or
  any stage below it (`filter`, `cache-writer`, `database-writer`, `api`, `web`).
- `bridge` kept producing normally (0 failures), so the raw feed kept arriving and
  consumer lag climbed the whole time the pipeline was stalled.
- One record was permanently lost (unrecoverable; see root cause).

## Timeline

1. `normalizer` reaches offset 43008608 on `fixm.raw` and cannot decode the batch.
2. The `kafka-go` group reader retries the same offset forever without surfacing an
   error, so the pod logs only startup lines and nothing else.
3. The stall is noticed by hand: downstream output stopped while `bridge` is healthy.
4. Diagnosis: the consumer group shows CURRENT-OFFSET frozen at 43008608 with LAG
   climbing; `envelopes_processed_total` is 0 (it never reached processing); a
   `kafka-console-consumer` read of that offset returns a CRC error, and
   `kafka-dump-log` confirms the batch is `isvalid: false`.
5. Recovery: scale `normalizer` to 0, reset the group offset to 43008609, scale
   back up. The backlog drains and the pipeline resumes.

## Root cause

The producer's CRC was valid on the way in (the broker validates on receive), so
the corruption happened **at rest**, after a good write:

```
stored crc = 247314664, computed crc = 2070435993   (isvalid: false)
```

The bytes on disk no longer matched their own checksum. The prime suspect is
**unstable host RAM**: `deploy/deploy.sh` already documents the build host
intermittently bit-flipping the Go compiler ("a build that doesn't crash could
still be silently bit-flipped"). Non-ECC RAM flips bits across every process on the
box, Kafka's page cache included.

Because `fixm.raw` is **replication-factor 1**, there was no healthy replica to
heal from, so the record was unrecoverable and had to be skipped.

**Why it was invisible:** `kafka-go`'s high-level group reader does not surface a
corrupt-batch error to the caller. `FetchMessage` returned neither a message nor an
error, it just retried the same offset. Hence no error logs and
`envelopes_processed_total` stuck at 0.

## Resolution (how it was recovered)

Skip past the poison record. Offsets can only be reset while the group has no active
member, so scale the consumer down first. Kafka's internal listener is
`kafka:29092` (there is no `localhost:9092` inside the pod).

First find the corrupt batch's `lastOffset` (skip to `lastOffset + 1`):

```bash
kubectl exec -n nas kafka-0 -- sh -c \
  '/opt/kafka/bin/kafka-dump-log.sh --files /var/lib/kafka/data/fixm.raw-0/<SEGMENT>.log 2>&1 \
   | grep -E "baseOffset: 4300860[0-9]"'
```

Then skip it:

```bash
# 1. stop the consumer
kubectl scale -n nas deploy/normalizer --replicas=0

# 2. confirm the group is empty ("has no active members")
kubectl exec -n nas kafka-0 -- /opt/kafka/bin/kafka-consumer-groups.sh \
  --bootstrap-server kafka:29092 --describe --group normalizer --members

# 3. seek past the corrupt batch
kubectl exec -n nas kafka-0 -- /opt/kafka/bin/kafka-consumer-groups.sh \
  --bootstrap-server kafka:29092 --group normalizer \
  --topic fixm.raw:0 --reset-offsets --to-offset 43008609 --execute

# 4. restart
kubectl scale -n nas deploy/normalizer --replicas=1
```

One record was lost, a sub-second blip for the live feed. If `kafka-dump-log` shows
several adjacent corrupt batches, skip to the first clean `baseOffset` past the
whole damaged run.

## Proposed fixes

- **Corrupt-batch auto-skip** in the Go consumers: classify the CRC/decode error,
  skip and dead-letter a marker, increment a metric, while still retrying transient
  errors. This removes the need for the manual recovery above.
  → [roadmap](../roadmap.md)
- **Alert on a stalled consumer** (`lag > 0` and offset flat): would have caught
  this in seconds instead of by hand.
  → [stalled-consumer post-mortem](stalled-consumer.md)
- **RF ≥ 3** across separate hosts so a bad copy self-heals.
- **Fix the host RAM** (memtest86+, then ECC): the underlying cause. XMP/EXPO
  disabled 2026-09-09.

## Lessons

- A single bad record with no auto-skip can freeze an entire stage indefinitely.
- The `kafka-go` group reader hides corrupt-batch errors, so "no error logs" does
  not mean "no error."
- The stall was invisible for lack of alerting, not lack of signal: consumer-group
  lag told the whole story once someone looked.
