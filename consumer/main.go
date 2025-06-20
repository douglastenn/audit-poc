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

	// Load schema
	schemaBytes, err := os.ReadFile("schema/audit_event.avsc")
	if err != nil {
		log.Fatalf("failed to read schema: %v", err)
	}
	schema, err := avro.Parse(string(schemaBytes))
	if err != nil {
		log.Fatalf("failed to parse avro schema: %v", err)
	}

	// OpenSearch client
	osClient, err := opensearch.NewClient(opensearch.Config{
		Addresses: []string{"http://localhost:9200"},
	})
	if err != nil {
		log.Fatalf("failed to create OpenSearch client: %v", err)
	}

	// AWS S3 client (via LocalStack)
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
		log.Fatalf("failed to load AWS config: %v", err)
	}
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	// Kafka consumer
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{"localhost:9092"},
		Topic:       "audit-events",
		GroupID:     "audit-consumer-group",
		StartOffset: kafka.FirstOffset,
		MinBytes:    1,
		MaxBytes:    10e6,
	})
	defer reader.Close()

	log.Println("🟢 Consumer ready. Waiting for messages...")

	for {
		m, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("error reading message: %v", err)
			continue
		}

		var event AuditEvent
		err = avro.Unmarshal(schema, m.Value, &event)
		if err != nil {
			log.Printf("⚠️ Failed to decode Avro: %v", err)
			continue
		}

		// Marshal to JSON for OS/S3
		jsonBytes, err := json.Marshal(event)
		if err != nil {
			log.Printf("⚠️ Failed to marshal event to JSON: %v", err)
			continue
		}

		// Index in OpenSearch
		res, err := osClient.Index("audit-events",
			bytes.NewReader(jsonBytes),
			osClient.Index.WithDocumentID(event.ID),
			osClient.Index.WithContext(ctx),
		)
		if err != nil {
			log.Printf("❌ OpenSearch index error: %v", err)
		} else {
			log.Printf("🔍 Indexed to OpenSearch: %s", event.ID)
			res.Body.Close()
		}

		// Save to S3 as cold storage
		s3Key := fmt.Sprintf("audit-events/%s.json", event.ID)
		_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String("audit-poc"),
			Key:    aws.String(s3Key),
			Body:   bytes.NewReader(jsonBytes),
		})
		if err != nil {
			log.Printf("❌ S3 upload error: %v", err)
		} else {
			log.Printf("📦 Uploaded to S3: %s", s3Key)
		}
	}
}
