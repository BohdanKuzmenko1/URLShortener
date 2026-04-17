package broker

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
)

type RedirectConsumer struct {
	reader *kafka.Reader
}

type RedirectEvent struct {
	URLId     int    `json:"url_id"`
	ClientIP  string `json:"client_ip"`
	Referer   string `json:"referer"`
	Country   string `json:"country"`
	UserAgent string `json:"user_agent"`
	CreatedAt int64  `json:"created_at"`
}

type BatchHandler func(ctx context.Context, events []RedirectEvent) error

type batchItem struct {
	events []RedirectEvent
	msgs   []kafka.Message
}

func NewRedirectConsumer() (*RedirectConsumer, error) {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		return nil, errors.New("KAFKA_BROKER environment variable not set")
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{broker},
		Topic:    "redirects",
		GroupID:  "stats-service",
		MinBytes: 1,
		MaxBytes: 10e6,
		MaxWait:  100 * time.Millisecond,
	})

	return &RedirectConsumer{reader: reader}, nil
}

func (c *RedirectConsumer) StartBatch(
	ctx context.Context,
	handle BatchHandler,
	batchSize int,
	flushInterval time.Duration,
) {
	log.Println("Kafka batch consumer started")

	batchCh := make(chan batchItem, 100)

	go func() {
		defer close(batchCh)

		current := batchItem{
			events: make([]RedirectEvent, 0, batchSize),
			msgs:   make([]kafka.Message, 0, batchSize),
		}

		ticker := time.NewTicker(flushInterval)
		defer ticker.Stop()

		flush := func() {
			if len(current.events) == 0 {
				return
			}
			b := batchItem{
				events: make([]RedirectEvent, len(current.events)),
				msgs:   make([]kafka.Message, len(current.msgs)),
			}
			copy(b.events, current.events)
			copy(b.msgs, current.msgs)
			batchCh <- b

			current.events = current.events[:0]
			current.msgs = current.msgs[:0]
		}

		for {
			select {
			case <-ctx.Done():
				flush()
				return
			case <-ticker.C:
				flush()
			default:
				msg, err := c.reader.FetchMessage(ctx)
				if err != nil {
					if ctx.Err() != nil {
						flush()
						return
					}
					log.Printf("[FETCH ERROR] %v", err)
					continue
				}

				var event RedirectEvent
				if err := json.Unmarshal(msg.Value, &event); err != nil {
					log.Printf("[UNMARSHAL ERROR] %v", err)
					_ = c.reader.CommitMessages(ctx, msg)
					continue
				}

				current.events = append(current.events, event)
				current.msgs = append(current.msgs, msg)

				if len(current.events) >= batchSize {
					flush()
				}
			}
		}
	}()

	for b := range batchCh {
		if err := handle(ctx, b.events); err != nil {
			log.Printf("[HANDLE ERROR] %v", err)
			continue
		}

		if err := c.reader.CommitMessages(ctx, b.msgs...); err != nil {
			log.Printf("[COMMIT ERROR] %v", err)
		}
	}
}

func (c *RedirectConsumer) Close() error {
	log.Println("closing kafka consumer")
	return c.reader.Close()
}
