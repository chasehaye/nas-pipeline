package com.naspipeline.bridge;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

/**
 * Bridges SWIM SFDPS to Kafka.
 *
 * This service provides the ingestion boundary between FAA SWIM SFDPS and
 * Kafka. FAA SWIM requires JMS for SFDPS communication, so this Java service
 * is responsible only for message retrieval and forwarding. Payload parsing
 * and transformation are performed by downstream processing services.
 */

@SpringBootApplication
public class BridgeApplication {

	public static void main(String[] args) {
		SpringApplication.run(BridgeApplication.class, args);
	}

}
