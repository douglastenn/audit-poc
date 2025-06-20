# 🕵️ Audit Logs PoC

Proof of Concept for an audit log pipeline using:

- ✅ Go + Avro
- 🦊 Kafka (Redpanda)
- 🪣 S3 (via LocalStack)
- 🔍 OpenSearch
- 🧠 Glue Schema Registry (emulated)

---

## 📦 Stack Overview

| Service        | Tool              | Port |
|----------------|-------------------|------|
| Kafka          | Redpanda          | 9092 |
| Object Storage | S3 via LocalStack | 4566 |
| Indexing       | OpenSearch        | 9200 |

---

## 🚀 How to Run

### 1. Start the infrastructure:

```bash
make up
```

### 2. Create Kafka topic and S3 bucket:

```bash
make create-topic audit-events
make create-bucket audit-poc
```

### 3. Start the consumer:

```bash
make run-consumer
```

### 4. Send events to Kafka:

```bash
make run-producer
```

---

## 🔍 Observability

### View Kafka messages:

```bash
make consume-topic
```

### Query OpenSearch:

```bash
make query-opensearch
```

### List uploaded files in S3:

```bash
make list-objects
make show-last-event
```

---

## 🧼 Cleanup

```bash
make down
```