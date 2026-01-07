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

// EventHandler is a function that processes an event.
type EventHandler func(ctx context.Context, event events.Event) error

// Subscriber handles listening for events from NATS.
type Subscriber struct {
	nc *nats.Conn
	js jetstream.JetStream
}

// NewSubscriber creates a new NATS subscriber.
// Validates that it reuses the same connection logic if possible,
// but for simplicity we create a new one or share the connection if refactored.
func NewSubscriber(url string) (*Subscriber, error) {
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

	return &Subscriber{nc: nc, js: js}, nil
}

// Subscribe registers a handler for a specific event subject pattern.
// It uses a persistent consumer (Durable) to ensure no messages are lost.
func (s *Subscriber) Subscribe(subject string, durableName string, handler EventHandler) error {
	ctx := context.Background()

	// Create Consumer
	consumer, err := s.js.CreateOrUpdateConsumer(ctx, "EVENTS", jetstream.ConsumerConfig{
		Durable:       durableName,
		FilterSubject: subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	// Consume Messages
	_, err = consumer.Consume(func(msg jetstream.Msg) {
		// Parse Event (Assuming Generic base for simplicity, or we check type)
		// For dynamic handling, we might need to unmarshal to a map first
		var payload map[string]interface{}
		if err := json.Unmarshal(msg.Data(), &payload); err != nil {
			log.Printf("Error unmarshalling event data: %v", err)
			msg.Nak()
			return
		}

		// Reconstruct Event Wrapper
		// NOTE: In a real system, you might want to embed the type in the payload or header
		// For now we assume the subject contains the type or we extract it
		event := events.BaseEvent{
			Type:       msg.Subject(), // rough approximation
			Data:       payload,
			OccurredAt: time.Now(), // If timestamp was in payload, extract it
		}

		// Execute Handler
		err := handler(context.Background(), event)
		if err != nil {
			log.Printf("Handler failed for event %s: %v", msg.Subject(), err)
			msg.Nak() // Retry
			return
		}

		msg.Ack()
	})

	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	log.Printf("Subscribed to %s with durable %s", subject, durableName)
	return nil
}

// Close closes the connection.
func (s *Subscriber) Close() {
	if s.nc != nil {
		s.nc.Close()
	}
}
