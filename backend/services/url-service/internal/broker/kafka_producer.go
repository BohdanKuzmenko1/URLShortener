package broker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

const topic = "redirects"

func NewProducer() (*Producer, error) {
	brokerAddr := os.Getenv("KAFKA_BROKER")
	if brokerAddr == "" {
		return nil, errors.New("KAFKA_BROKER environment variable not set")
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokerAddr),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		MaxAttempts:  3,
		Async:        true,
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,

		Completion: func(messages []kafka.Message, err error) {
			if err != nil {
				logrus.Errorf("kafka async send failed for %d messages: %v", len(messages), err)
			}
		},
	}

	return &Producer{writer: writer}, nil
}

type Producer struct {
	writer *kafka.Writer
}

func (p *Producer) SendRedirect(ctx context.Context, redirectEvent RedirectEvent, slug string) error {
	if p.writer == nil {
		return errors.New("kafka writer is not initialized")
	}

	value, err := json.Marshal(redirectEvent)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(slug),
		Value: value,
	}

	p.writer.WriteMessages(ctx, msg)

	return nil
}

func CreateTopic(broker string, topic string, partitions int) error {
	conn, err := kafka.Dial("tcp", broker)
	if err != nil {
		return err
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return err
	}

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return err
	}
	defer controllerConn.Close()

	err = controllerConn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     partitions,
		ReplicationFactor: 1,
	})
	if err != nil && !errors.Is(err, kafka.TopicAlreadyExists) {
		return err
	}

	return nil
}
