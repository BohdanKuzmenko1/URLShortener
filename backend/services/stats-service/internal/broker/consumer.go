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

// RedirectConsumer reads redirect events from a Kafka topic
// and processes them in batches.
type RedirectConsumer struct {
	reader *kafka.Reader
}

// RedirectEvent represents a single redirect captured by the Stats Service.
// It is deserialized from a Kafka message payload.
type RedirectEvent struct {
	URLId     int    `json:"url_id"`
	ClientIP  string `json:"client_ip"`
	Referer   string `json:"referer"`
	Country   string `json:"country"`
	UserAgent string `json:"user_agent"`
	CreatedAt int64  `json:"created_at"`
}

// BatchHandler is a function that processes a slice of redirect events.
// It is called once per flushed batch by the consumer goroutine.
// If it returns an error, the batch is dropped and Kafka offsets are not committed.
type BatchHandler func(ctx context.Context, events []RedirectEvent) error

// batchItem holds a snapshot of accumulated events and their corresponding
// Kafka messages. Both slices are kept in sync: events[i] was decoded from msgs[i].
// msgs is needed to commit Kafka offsets after the batch is successfully handled.
type batchItem struct {
	events []RedirectEvent
	msgs   []kafka.Message
}

// NewRedirectConsumer creates a Kafka reader subscribed to the "redirects" topic.
// The reader joins the "stats-service" consumer group, so Kafka tracks offsets
// per instance and distributes partitions across multiple replicas automatically.
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
		// MaxWait caps how long FetchMessage blocks waiting for new data.
		// Keeping it short (100ms) ensures the ticker-based flush fires on time
		// even when the topic is quiet.
		MaxWait: 100 * time.Millisecond,
	})

	return &RedirectConsumer{reader: reader}, nil
}

// StartBatch runs the batch consumer loop. It blocks until ctx is cancelled.
//
// Internally it spawns a producer goroutine that reads from Kafka and accumulates
// events into a buffer. The buffer is flushed — copied into batchCh — when either:
//   - the buffer reaches batchSize (throughput trigger), or
//   - flushInterval elapses (latency trigger, ensures progress at low traffic).
//
// The main goroutine (consumer side) reads from batchCh, calls handle, and only
// then commits Kafka offsets. This provides at-least-once delivery: if the process
// crashes after handle succeeds but before the commit, the batch will be replayed.
func (c *RedirectConsumer) StartBatch(
	ctx context.Context,
	handle BatchHandler,
	batchSize int,
	flushInterval time.Duration,
) {
	log.Println("Kafka batch consumer started")

	// batchCh decouples the producer goroutine (reads Kafka, builds batches)
	// from the consumer goroutine (calls handle, commits offsets).
	// A buffer of 100 prevents the producer from stalling if handle is slow.
	batchCh := make(chan batchItem, 100)

	// Producer goroutine: accumulates messages and flushes complete batches.
	go func() {
		// Closing batchCh signals the consumer loop below to exit
		// once all pending batches have been drained.
		defer close(batchCh)

		// current is the live accumulation buffer. It is allocated once with
		// cap=batchSize and reused across flushes via [:0] to avoid GC pressure.
		current := batchItem{
			events: make([]RedirectEvent, 0, batchSize),
			msgs:   make([]kafka.Message, 0, batchSize),
		}

		ticker := time.NewTicker(flushInterval)
		defer ticker.Stop()

		// flush drains the current accumulation buffer into batchCh.
		// A full copy is made so the consumer goroutine and this goroutine
		// never share the same underlying array.
		// The buffer is reset via [:0] to preserve its capacity for reuse.
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
			// Graceful shutdown: flush whatever is buffered before exiting
			// so no events are silently dropped when the context is cancelled.
			case <-ctx.Done():
				flush()
				return

			// Time-based flush: guarantees forward progress when traffic is low
			// and the buffer never reaches batchSize on its own.
			case <-ticker.C:
				flush()

			// Default runs when neither signal is ready, keeping the loop
			// as tight as possible without blocking on the select itself.
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
					// A malformed message is unrecoverable: commit it immediately
					// so it is not re-delivered, and move on.
					log.Printf("[UNMARSHAL ERROR] %v", err)
					_ = c.reader.CommitMessages(ctx, msg)
					continue
				}

				current.events = append(current.events, event)
				current.msgs = append(current.msgs, msg)

				// Size-based flush: keeps individual batches bounded
				// and prevents unbounded memory growth under high traffic.
				if len(current.events) >= batchSize {
					flush()
				}
			}
		}
	}()

	// Consumer loop: runs on the calling goroutine.
	// Ranges over batchCh until the producer closes it.
	for b := range batchCh {
		if err := handle(ctx, b.events); err != nil {
			// Dropping the batch on error means these events will not be
			// retried unless Kafka replays them (e.g. after a restart before commit).
			log.Printf("[HANDLE ERROR] %v", err)
			continue
		}

		// Offsets are committed only after a successful handle call.
		// This ensures at-least-once semantics: a crash here causes a replay,
		// not a silent loss.
		if err := c.reader.CommitMessages(ctx, b.msgs...); err != nil {
			log.Printf("[COMMIT ERROR] %v", err)
		}
	}
}

// Close shuts down the underlying Kafka reader and releases its resources.
// It should be called after StartBatch returns (e.g. via defer).
func (c *RedirectConsumer) Close() error {
	log.Println("closing kafka consumer")
	return c.reader.Close()
}
