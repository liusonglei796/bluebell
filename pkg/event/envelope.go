package event

import (
	"encoding/json"
	"time"
)

// Envelope wraps all domain events in a standard, auditable container.
type Envelope[T any] struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"`
	Timestamp int64  `json:"timestamp"`
	ActorID   int64  `json:"actor_id"`
	Producer  string `json:"producer"`
	Version   string `json:"version"`
	Payload   T      `json:"payload"`
}

// NewEnvelope constructs a standard envelope with current timestamp and defaults.
func NewEnvelope[T any](eventID string, eventType string, actorID int64, producer string, payload T) *Envelope[T] {
	return &Envelope[T]{
		EventID:   eventID,
		EventType: eventType,
		Timestamp: time.Now().UTC().UnixMilli(),
		ActorID:   actorID,
		Producer:  producer,
		Version:   "v1",
		Payload:   payload,
	}
}

// RawEnvelope is used for initial inspection and dynamic deserialization by consumers.
type RawEvent struct {
	EventID   string          `json:"event_id"`
	EventType string          `json:"event_type"`
	Timestamp int64           `json:"timestamp"`
	ActorID   int64           `json:"actor_id"`
	Producer  string          `json:"producer"`
	Version   string          `json:"version"`
	Payload   json.RawMessage `json:"payload"`
}
