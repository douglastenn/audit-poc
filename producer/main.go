package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/hamba/avro"
	"github.com/segmentio/kafka-go"
)

type AuditEvent struct {
	ID        string  `avro:"id"`
	Action    string  `avro:"action"`
	ContactID *string `avro:"contactId"`
	CreatedAt string  `avro:"createdAt"`
	//Details   interface{} `avro:"details"`
}

func main() {
	schemaBytes, err := os.ReadFile("schema/audit_event.avsc")
	if err != nil {
		log.Fatalf("failed to read schema file: %v", err)
	}

	schema, err := avro.Parse(string(schemaBytes))
	if err != nil {
		log.Fatalf("failed to parse Avro schema: %v", err)
	}

	contactID := "c-456"

	event := AuditEvent{
		ID:        "evt-123",
		Action:    "edited",
		ContactID: &contactID,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		//Details: map[string]interface{}{
		//	"field":    "email",
		//	"oldValue": "foo@bar.com",
		//	"newValue": "bar@foo.com",
		//},
	}

	data, err := avro.Marshal(schema, event)
	if err != nil {
		log.Fatalf("failed to serialize: %v", err)
	}

	writer := kafka.Writer{
		Addr:     kafka.TCP("localhost:9092"),
		Topic:    "audit-events",
		Balancer: &kafka.LeastBytes{},
	}

	err = writer.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte("event"),
			Value: data,
		})
	if err != nil {
		log.Fatalf("failed to write message: %v", err)
	}

	log.Println("✅ Message sent to Kafka.")
}
