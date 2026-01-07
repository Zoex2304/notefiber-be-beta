package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ai-notetaking-be/pkg/events"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Publisher handles sending events to the NATS bus.
type Publisher struct {
	nc *nats.Conn
	js jetstream.JetStream
}

// NewPublisher creates a new NATS publisher.
func NewPublisher(url string) (*Publisher, error) {
	nc, err := nats.Connect(url,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(5),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Ensure the "EVENTS" stream exists
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      "EVENTS",
		Subjects:  []string{"events.>"},
		Storage:   jetstream.FileStorage,
		Retention: jetstream.WorkQueuePolicy, // Or LimitsPolicy depending on need
	})
	if err != nil {
		log.Printf("Warn: Failed to ensure stream 'EVENTS': %v", err)
		// Don't fail hard here, maybe it already exists or NATS isn't ready
	}

	return &Publisher{nc: nc, js: js}, nil
}

// Publish sends an event to NATS.
func (p *Publisher) Publish(ctx context.Context, event events.Event) error {
	data, err := json.Marshal(event.Payload())
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	subject := fmt.Sprintf("events.%s", event.EventType())

	// Use JetStream Publish
	// Note: We use the context for timeout
	_, err = p.js.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish event to subject %s: %w", subject, err)
	}

	return nil
}

// Close closes the NATS connection.
func (p *Publisher) Close() {
	if p.nc != nil {
		p.nc.Close()
	}
}
