package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hamba/avro"
	"github.com/opensearch-project/opensearch-go"
	"github.com/segmentio/kafka-go"
)

type AuditEvent struct {
	ID        string  `avro:"id" json:"id"`
	Action    string  `avro:"action" json:"action"`
	ContactID *string `avro:"contactId" json:"contactId"`
	CreatedAt string  `avro:"createdAt" json:"createdAt"`
}

func main() {
	ctx := context.Background()

	// Load Avro schema from file
	schemaBytes, err := os.ReadFile("schema/audit_event.avsc")
	if err != nil {
		log.Fatalf("❌ Failed to read schema: %v", err)
	}
	schema, err := avro.Parse(string(schemaBytes))
	if err != nil {
		log.Fatalf("❌ Failed to parse Avro schema: %v", err)
	}

	// OpenSearch client
	osClient, err := opensearch.NewClient(opensearch.Config{
		Addresses: []string{"http://localhost:9200"},
	})
	if err != nil {
		log.Fatalf("❌ Failed to create OpenSearch client: %v", err)
	}

	// AWS S3 client (LocalStack)
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:           "http://localhost:4566",
					SigningRegion: "us-east-1",
				}, nil
			}),
		),
	)
	if err != nil {
		log.Fatalf("❌ Failed to load AWS config: %v", err)
	}
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	// Kafka consumer (Redpanda)
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{"localhost:9092"},
		Topic:       "audit-events",
		GroupID:     "audit-consumer-group",
		StartOffset: kafka.FirstOffset,
		MinBytes:    1,
		MaxBytes:    10e6,
	})
	defer reader.Close()

	log.Println("🟢 Kafka consumer started. Waiting for messages...")

	for {
		m, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("⚠️ Error reading message: %v", err)
			continue
		}

		// Decode Avro
		var event AuditEvent
		err = avro.Unmarshal(schema, m.Value, &event)
		if err != nil {
			log.Printf("⚠️ Failed to decode Avro: %v", err)
			continue
		}

		// Index to OpenSearch
		jsonBytes, err := json.Marshal(event)
		if err != nil {
			log.Printf("❌ Failed to marshal JSON: %v", err)
			continue
		}

		res, err := osClient.Index("audit-events",
			bytes.NewReader(jsonBytes),
			osClient.Index.WithDocumentID(event.ID),
			osClient.Index.WithContext(ctx),
		)
		if err != nil {
			log.Printf("❌ OpenSearch index error: %v", err)
		} else {
			log.Printf("🔍 Indexed in OpenSearch: %s", event.ID)
			res.Body.Close()
		}

		// Save Avro to S3
		avroBytes, err := avro.Marshal(schema, event)
		if err != nil {
			log.Printf("❌ Failed to serialize Avro: %v", err)
			continue
		}

		s3Key := fmt.Sprintf("audit-events/%s.avro", event.ID)

		_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String("audit-poc"),
			Key:    aws.String(s3Key),
			Body:   bytes.NewReader(avroBytes),
		})
		if err != nil {
			log.Printf("❌ S3 upload error: %v", err)
		} else {
			log.Printf("📦 Uploaded to S3: %s", s3Key)
		}
	}
}
