package kafka

import (
	"strconv"

	"github.com/segmentio/kafka-go"
)


func EnsureTopic(
	brokers string,
	topic string,
	partitions int,
	replicationFactor int,
) error {

	conn, err := kafka.Dial(
		"tcp",
		brokers,
	)

	if err != nil {
		return err
	}

	defer conn.Close()


	controller, err := conn.Controller()

	if err != nil {
		return err
	}


	controllerConn, err := kafka.Dial(
		"tcp",
		controller.Host+":"+strconv.Itoa(controller.Port),
	)

	if err != nil {
		return err
	}

	defer controllerConn.Close()


	// CreateTopics is idempotent: kafka-go swallows the broker's
	// TopicAlreadyExists (code 36) and returns nil, so creating an existing
	// topic is a harmless no-op — no special-casing needed here.
	return controllerConn.CreateTopics(
		kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     partitions,
			ReplicationFactor: replicationFactor,
		},
	)
}