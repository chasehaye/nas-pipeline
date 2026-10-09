package com.naspipeline.bridge;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.CommandLineRunner;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.stereotype.Component;

import java.io.BufferedReader;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Duration;
import java.time.Instant;
import java.util.List;
import java.util.concurrent.atomic.AtomicLong;
import java.util.stream.Stream;

/**
 * Dev-only replay source, active when ISDEV=true (mutually exclusive with the
 * Solace FixmListener). It reads the committed seed and republishes each record
 * to fixm.raw in a loop, standing in for the live SWIM/Solace feed so the whole
 * pipeline can run locally with no SWIM access and no Solace credentials.
 *
 * The seed is produced offline (capture real fixm.raw, then run it through the
 * LADD filter), so it is safe to commit and contains no blocked aircraft. Its
 * records carry the same raw FIXM shape the bridge normally publishes and are
 * separated by a marker line, matching the capture/filter output format.
 *
 * The seed is split across several files in a directory (GitHub caps a single
 * file at 100 MB), so this reads every file in that directory in sorted name
 * order and treats them as one continuous stream.
 */
@Component
@ConditionalOnProperty(name = "isdev", havingValue = "true")
public class ReplayRunner implements CommandLineRunner {

    private static final Logger log = LoggerFactory.getLogger(ReplayRunner.class);

    // Record boundary written by the capture/filter tooling.
    private static final String SEPARATOR = "-----RECORD-----";

    private final KafkaTemplate<String, String> kafka;
    private final String topic;
    private final Path seedDir;
    private final long delayMs;
    private final boolean loop;

    public ReplayRunner(KafkaTemplate<String, String> kafka,
                        @Value("${kafka.topic.raw:fixm.raw}") String topic,
                        @Value("${replay.dir:seed-data}") String seedDir,
                        @Value("${replay.delay-ms:250}") long delayMs,
                        @Value("${replay.loop:true}") boolean loop) {
        this.kafka = kafka;
        this.topic = topic;
        this.seedDir = Path.of(seedDir);
        this.delayMs = delayMs;
        this.loop = loop;
    }

    @Override
    public void run(String... args) throws Exception {
        if (!Files.isDirectory(seedDir)) {
            throw new IllegalStateException(
                    "replay seed dir not found: " + seedDir.toAbsolutePath()
                            + " (set REPLAY_DIR or run from the directory holding the seed)");
        }

        List<Path> files = seedFiles();
        if (files.isEmpty()) {
            throw new IllegalStateException("no seed files in " + seedDir.toAbsolutePath());
        }

        log.info("ISDEV replay mode: streaming {} seed file(s) from '{}' to topic '{}' (delayMs={}, loop={})",
                files.size(), seedDir.toAbsolutePath(), topic, delayMs, loop);

        AtomicLong sent = new AtomicLong();
        Instant startedAt = Instant.now();

        do {
            for (Path file : files) {
                replayFile(file, sent, startedAt);
            }
            if (loop) {
                log.info("reached end of seed ({} files), looping", files.size());
            }
        } while (loop);

        log.info("replay complete, sent {} records", sent.get());
    }

    // All regular files in the seed dir, sorted by name so the split parts
    // (dev-stream-00.xml, -01.xml, ...) replay in order.
    private List<Path> seedFiles() throws Exception {
        try (Stream<Path> s = Files.list(seedDir)) {
            return s.filter(Files::isRegularFile).sorted().toList();
        }
    }

    // Streams one file a record at a time (the seed is large), splitting on the
    // separator line so memory stays flat regardless of file size.
    private void replayFile(Path file, AtomicLong sent, Instant startedAt) throws Exception {
        try (BufferedReader reader = Files.newBufferedReader(file, StandardCharsets.UTF_8)) {
            StringBuilder buffer = new StringBuilder();
            String line;
            while ((line = reader.readLine()) != null) {
                if (line.equals(SEPARATOR)) {
                    publishRecord(buffer, sent, startedAt);
                    buffer.setLength(0);
                } else {
                    if (buffer.length() > 0) {
                        buffer.append('\n');
                    }
                    buffer.append(line);
                }
            }
            // Flush a trailing record that was not followed by a separator.
            publishRecord(buffer, sent, startedAt);
        }
    }

    private void publishRecord(StringBuilder buffer, AtomicLong sent, Instant startedAt)
            throws InterruptedException {
        String record = buffer.toString().strip();
        if (record.isEmpty()) {
            return;
        }

        // Hook for forward-dating timestamps so the live map does not go stale.
        // Currently a no-op; see rewriteTimestamps.
        record = rewriteTimestamps(record);

        // Mirror the bridge's contract: no key (one record carries many GUFIs).
        kafka.send(topic, record);

        long n = sent.incrementAndGet();
        if (n % 1000 == 0) {
            long seconds = Math.max(1, Duration.between(startedAt, Instant.now()).toSeconds());
            log.info("replayed {} records, {} rec/sec avg", n, n / seconds);
        }

        if (delayMs > 0) {
            Thread.sleep(delayMs);
        }
    }

    /**
     * TODO (next step): shift the position/flight timestamps forward to ~now on
     * each pass so cache-writer TTLs stay alive and the map shows current
     * positions. Until then the replayed data keeps its original capture-time
     * timestamps, which is fine for exercising normalizer/filter/writers but
     * will look stale on the live map. The fields that matter are the flight
     * 'timestamp' attribute and the position 'positionTime' /
     * 'targetPositionTime'.
     */
    private String rewriteTimestamps(String record) {
        return record;
    }
}
