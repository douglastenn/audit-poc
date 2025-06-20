# -------------------------------------
# 📦 Environment
# -------------------------------------
REDPANDA_CONTAINER = audit-poc-redpanda-1
LOCALSTACK_CONTAINER = audit-poc-localstack-1
TOPIC_NAME = audit-events
BUCKET_NAME = audit-poc
OPENSEARCH_URL = http://localhost:9200

.PHONY: help up down logs \
	create-topic list-topics consume-topic \
	create-bucket list-buckets list-objects show-last-event \
	query-opensearch list-opensearch-indices \
	run-producer run-consumer

# -------------------------------------
# 🔧 Docker Infrastructure
# -------------------------------------

up: ## 🚀 Start Docker Compose
	docker-compose up -d

down: ## 🧯 Stop Docker Compose
	docker-compose down

logs: ## 📋 View logs from containers
	docker-compose logs -f

# -------------------------------------
# 🦊 Kafka / Redpanda
# -------------------------------------

create-topic: ## 🧵 Create Kafka topic: $(TOPIC_NAME)
	docker exec -it $(REDPANDA_CONTAINER) rpk topic create $(TOPIC_NAME)

list-topics: ## 📜 List all Kafka topics
	docker exec -it $(REDPANDA_CONTAINER) rpk topic list

consume-topic: ## 👂 Consume messages from topic
	docker exec -it $(REDPANDA_CONTAINER) rpk topic consume $(TOPIC_NAME)

# -------------------------------------
# 🪣 S3 / LocalStack
# -------------------------------------

create-bucket: ## 🪣 Create S3 bucket: $(BUCKET_NAME)
	docker exec -it $(LOCALSTACK_CONTAINER) awslocal s3 mb s3://$(BUCKET_NAME)

list-buckets: ## 📦 List all S3 buckets
	docker exec -it $(LOCALSTACK_CONTAINER) awslocal s3 ls

list-objects: ## 🗂️  List objects in your bucket
	docker exec -it $(LOCALSTACK_CONTAINER) awslocal s3 ls s3://$(BUCKET_NAME)

show-last-event: ## 📄 Show contents of a sample audit event from S3
	docker exec -it $(LOCALSTACK_CONTAINER) awslocal s3 cp s3://$(BUCKET_NAME)/$(TOPIC_NAME)/evt-123.json -

# -------------------------------------
# 🔍 OpenSearch
# -------------------------------------

query-opensearch: ## 🔎 List all docs from OpenSearch index
	curl -X GET "$(OPENSEARCH_URL)/$(TOPIC_NAME)/_search?pretty" \
		-H "Content-Type: application/json" \
		-d '{"query":{"match_all":{}}}'

list-opensearch-indices: ## 📂 List all OpenSearch indices
	curl -X GET "$(OPENSEARCH_URL)/_cat/indices?v"

# -------------------------------------
# 👨‍💻 Go App
# -------------------------------------

run-producer: ## ▶️  Run Go producer
	go run producer/main.go

run-consumer: ## 👂 Run Go Kafka consumer
	go run consumer/main.go

# -------------------------------------
# 📘 Help
# -------------------------------------

help: ## 🆘 Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## ' Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "🛠️  \033[36m%-25s\033[0m %s\n", $$1, $$2}'