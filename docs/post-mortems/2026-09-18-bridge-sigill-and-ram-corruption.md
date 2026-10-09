# Post-mortem: bridge SIGILL + recurring RAM corruption (2026-09-18)

## Summary

Two overlapping failures took the pipeline down. (1) A fresh Kafka bit-corruption
stalled `normalizer` (same class as the [2026-09-03 incident](2026-09-03-corrupt-batch.md)).
(2) `bridge` began crash-looping with `SIGILL` at JVM startup, so ingestion stopped.
The two had **different root causes**: the corruption is the host's bad non-ECC
RAM, and the bridge crash is a new-glibc-vs-old-runc incompatibility. They surfaced
together because a single reboot (very likely RAM-induced) triggered both. Recovery
also exposed a chronic problem: Postgres runs on an SMR RAID that collapses under
sustained writes.

## Impact

- `normalizer` frozen at `fixm.raw` offset 82679154 with a ~791k message backlog.
- `bridge` crash-looping, so no new SWIM data entering the pipeline.
- Whole pipeline down for the duration.

## Timeline

1. `normalizer` wedged on a run of corrupt records (`fixm.raw` offsets 82679153,
   82679154, 82679155 all `isvalid: false`, bit-flipped at rest). Lag grew while
   `bridge` was still producing.
2. A reboot/restart (this box has **77 unsafe shutdowns**) restarted `bridge` onto
   its Temurin 25 image, which had a **latent** glibc bug. `bridge` then SIGILL'd
   on every fresh start.
3. Long diagnosis of the SIGILL, ruled out (in order): image corruption, RAM,
   Temurin version, seccomp, glibc AVX ifunc, and kernel state (it survived a full
   reboot). Proved the identical image runs fine under `docker run` and
   `k3s ctr run`, only the kubelet pod crashed.
4. Captured `/tmp/hs_err_pid*.log`: the crash is in glibc `pthread_cond_timedwait`
   (a `futex` syscall) on the JVM's C2 compiler thread, at ~16ms into startup.
5. Fixed bridge by pinning to **Temurin 21 on Ubuntu jammy** (old glibc) plus
   `pom.xml` `java.version 21`. Bridge boots and produces again.
6. Skipped `normalizer` past the corrupt run to offset **82679156** (first clean
   batch); it began draining the backlog.
7. Recovery surfaced climbing `database-writer` batch latency: `iostat` showed the
   Postgres SMR RAID (`md1`) pinned at 100% util but only ~3 MB/s with ~134ms
   `fsync` latency, while the NVMe sat at ~0.8% util.

## Root causes

1. **Kafka corruption:** bad **non-ECC RAM** flipping bytes at rest. SMART on the
   NVMe is clean (`Media and Data Integrity Errors: 0`), so it is not the disk. RF=1
   meant there was no healthy replica to heal from, so recovery was a manual skip.
   Same underlying fault as 2026-09-03.
2. **Bridge SIGILL:** the very new glibc (Temurin 25 / Ubuntu 26.04 base) issues a
   `futex` syscall variant that **k3s's older bundled runc/containerd mishandles**,
   so glibc's `futex_fatal_error()` traps with an illegal instruction. Deterministic,
   and only under kubelet-launched pods (the host's own newer Docker runc handles it,
   which is why `docker run` worked). The bug is **latent**: it only bites on a fresh
   container start, so bridge "ran fine for weeks" until a restart.
3. **Postgres on SMR (chronic, surfaced during recovery):** the DB data dir lives on
   `/mnt/drive2`, an SMR RAID1 that collapses under sustained writes. The backlog
   catch-up is exactly that workload.

The common trigger tying (1) and (2) together: a bad-RAM-induced crash/reboot both
corrupted records at rest and forced bridge to restart into its latent bug.

## Resolution

- **normalizer:** `kafka-dump-log` on the segment to find the corrupt run, then
  scale to 0, `reset-offsets --to-offset 82679156`, scale to 1. Three records lost
  (a sub-second blip).
- **bridge:** `bridge/Dockerfile` to `eclipse-temurin:21-jdk-jammy` /
  `21-jre-jammy`, `bridge/pom.xml` to `<java.version>21</java.version>` (JDK 21
  cannot compile `release 25`), rebuild, import into k3s, roll out.

## Hardening plan

**Hardware (the actual cause of the corruption and the reboots):**
- Run **memtest86+**. The bad RAM is the source of both the data corruption and the
  77 unsafe shutdowns.
- Move to **ECC RAM**. The Ryzen 9 5900X supports ECC on a compatible board; then
  EDAC/`rasdaemon` can detect, correct, and predict a failing DIMM before it
  corrupts anything. Non-ECC leaves memory bit-flips completely invisible to the OS.

**Storage:**
- **Move Postgres off the SMR RAID onto the NVMe** (the NVMe sits ~idle). See the
  migration steps recorded with this incident.
- Consider **ZFS/btrfs** for data drives so at-rest corruption is detected via
  block checksums (note: fully trustworthy only with ECC, since a pre-checksum RAM
  flip is stored faithfully).

**Data resilience:**
- **Kafka RF >= 3** across separate brokers/hosts so a corrupt copy self-heals
  instead of needing a manual offset skip.
- **Corrupt-batch auto-skip** in the Go consumers (classify the CRC/decode error,
  skip + dead-letter a marker + metric) so one bad record degrades gracefully
  instead of freezing a stage. See [roadmap](../roadmap.md).

**Ops / observability:**
- **Alert on a stalled consumer** (`lag > 0` and current-offset flat) so this pages
  in seconds instead of a manual hunt.
- **Update k3s** (newer bundled runc) so the bridge Temurin pin can be reverted to a
  current JDK.
- `rasdaemon` for CPU/MCE monitoring, and regular `pg_dumpall` backups.

## Lessons

- **Co-occurring failures can have different root causes.** Here, one bad-RAM reboot
  surfaced two unrelated latent problems. We wasted time assuming a single cause.
- **"Works under docker/ctr, SIGILLs only as a k8s pod" = a runtime (runc/syscall)
  mismatch, not the image or the hardware.** Capture the `hs_err` log early; it named
  the futex frame and ended the guessing.
- **Latent version bugs lurk until the next restart.** Bridge's Temurin 25 bug was
  present for weeks and only bit when a reboot restarted it.
- **Kafka's CRC was the only thing that caught the corruption** (no ECC, RF=1).
  End-to-end checksums were doing ECC's job; add the hardware layer.
- **SMR drives plus a database is a pathological pairing** under sustained writes.
