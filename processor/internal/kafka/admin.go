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


	err = controllerConn.CreateTopics(
		kafka.TopicConfig{
			Topic: topic,
			NumPartitions: partitions,
			ReplicationFactor: replicationFactor,
		},
	)

	if err != nil && err.Error() != "topic already exists" {
		return err
	}

	return nil
}