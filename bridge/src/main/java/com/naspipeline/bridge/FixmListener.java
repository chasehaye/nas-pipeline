package com.naspipeline.bridge;

import jakarta.jms.BytesMessage;
import jakarta.jms.Message;
import jakarta.jms.TextMessage;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.jms.annotation.JmsListener;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.stereotype.Component;

import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.time.Instant;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicLong;


@Component
public class FixmListener {

    private static final Logger log = LoggerFactory.getLogger(FixmListener.class);

    private final KafkaTemplate<String, String> kafka;
    private final String topic;

    private final AtomicLong count = new AtomicLong();
    private final AtomicLong bytes = new AtomicLong();
    private final AtomicLong failures = new AtomicLong();
    private final Instant startedAt = Instant.now();

    public FixmListener(KafkaTemplate<String, String> kafka,
                        @Value("${kafka.topic.raw:fixm.raw}") String topic) {
        this.kafka = kafka;
        this.topic = topic;
        log.info("publishing raw FIXM to topic '{}'", topic);
    }

    @JmsListener(
            destination = "${solace.queue}",
            containerFactory = "jmsListenerContainerFactory"
    )
    public void onMessage(Message message) throws Exception {

        String body = extractBody(message);

        // No key. One JMS message carries many flights with different GUFIs,
        // so there is no single sensible partition key at this layer. The Go
        // consumer explodes the collection and re-keys per flight if needed.
        try {
            kafka.send(topic, body).get(30, TimeUnit.SECONDS);
        } catch (Exception e) {
            // Do NOT acknowledge. Solace keeps the message on the queue and
            // will redeliver it. Better to reprocess a duplicate than to
            // silently drop data.
            failures.incrementAndGet();
            log.error("kafka send failed, leaving message unacknowledged for redelivery", e);
            throw e;
        }

        // Only now is the message durably somewhere else. Acknowledging
        // before this point would mean a crash between send and ack loses
        // the message permanently: gone from Solace, never in Kafka.
        message.acknowledge();

        long n = count.incrementAndGet();
        long b = bytes.addAndGet(body.length());

        if (n % 1000 == 0) {
            long seconds = Math.max(1, Duration.between(startedAt, Instant.now()).toSeconds());
            log.info("published {} messages, {} MB, {} msg/sec avg, {} failures",
                    n, b / 1_048_576, n / seconds, failures.get());
        }
    }

    private String extractBody(Message message) throws Exception {
        if (message instanceof TextMessage text) {
            return text.getText();
        }
        if (message instanceof BytesMessage bytesMessage) {
            byte[] buf = new byte[(int) bytesMessage.getBodyLength()];
            bytesMessage.readBytes(buf);
            return new String(buf, StandardCharsets.UTF_8);
        }
        throw new IllegalStateException(
                "unhandled JMS message type: " + message.getClass().getName());
    }
}