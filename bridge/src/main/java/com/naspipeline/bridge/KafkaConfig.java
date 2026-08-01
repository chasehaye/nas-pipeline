package com.naspipeline.bridge;

import org.apache.kafka.clients.producer.ProducerConfig;
import org.apache.kafka.common.serialization.StringSerializer;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.kafka.core.DefaultKafkaProducerFactory;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.kafka.core.ProducerFactory;

import java.util.HashMap;
import java.util.Map;

/**
 * Explicit producer setup. Spring Boot 4 moved autoconfiguration into
 * per-technology modules, so having spring-kafka on the classpath is no
 * longer enough to get a KafkaTemplate for free. Declaring it here is a
 * few more lines but makes every setting visible in one place.
 */
@Configuration
public class KafkaConfig {

    @Value("${spring.kafka.bootstrap-servers:localhost:9092}")
    private String bootstrapServers;

    @Bean
    public ProducerFactory<String, String> producerFactory() {
        Map<String, Object> props = new HashMap<>();

        props.put(ProducerConfig.BOOTSTRAP_SERVERS_CONFIG, bootstrapServers);
        props.put(ProducerConfig.KEY_SERIALIZER_CLASS_CONFIG, StringSerializer.class);
        props.put(ProducerConfig.VALUE_SERIALIZER_CLASS_CONFIG, StringSerializer.class);

        // XML compresses roughly 10:1. Highest-leverage setting here given
        // the raw feed runs well over 100 GB/day.
        props.put(ProducerConfig.COMPRESSION_TYPE_CONFIG, "zstd");

        // Wait for in-sync replicas to confirm before send() completes.
        // Same as acks=1 on a single broker, but correct as you grow.
        props.put(ProducerConfig.ACKS_CONFIG, "all");
        props.put(ProducerConfig.RETRIES_CONFIG, 5);

        // Dedupes the producer's own retries at the broker, so an ambiguous
        // timeout followed by a retry doesn't write the message twice.
        props.put(ProducerConfig.ENABLE_IDEMPOTENCE_CONFIG, true);

        // FIXM collections reach ~370 KB. Leave plenty of headroom.
        props.put(ProducerConfig.MAX_REQUEST_SIZE_CONFIG, 10_485_760);

        // A short linger lets the producer batch, which noticeably improves
        // the compression ratio on payloads this size.
        props.put(ProducerConfig.LINGER_MS_CONFIG, 20);

        return new DefaultKafkaProducerFactory<>(props);
    }

    @Bean
    public KafkaTemplate<String, String> kafkaTemplate(ProducerFactory<String, String> pf) {
        return new KafkaTemplate<>(pf);
    }
}