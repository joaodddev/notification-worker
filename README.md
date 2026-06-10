# Notification Worker

A concurrent and persistent webhook processing service built with Go.

This project implements an asynchronous notification queue capable of receiving webhook jobs, persisting them in PostgreSQL, and processing them through a configurable worker pool with graceful shutdown support.

It was designed to demonstrate concurrency patterns, reliable job processing, and backend architecture concepts commonly found in production systems.

---

## ✨ Features

* 🚀 Asynchronous webhook processing
* 🗄️ Persistent job queue backed by PostgreSQL
* 👷 Concurrent worker pool using goroutines
* 🔒 Safe job dequeue with `FOR UPDATE SKIP LOCKED`
* 🔄 Graceful shutdown support
* 📡 HTTP API for job creation
* 🐳 Docker & Docker Compose ready
* ⚡ Modular architecture following Go best practices

---

## 🏗️ Architecture

```text
                Client
                   │
            POST /jobs
                   │
                   ▼
        ┌────────────────────┐
        │      API Server    │
        └─────────┬──────────┘
                  │
                  ▼
        ┌────────────────────┐
        │     PostgreSQL     │
        │   status=pending   │
        └─────────┬──────────┘
                  │
         Polling Dispatcher
             (every 2s)
                  │
                  ▼
          Buffered Job Channel
                  │
      ┌───────────┼───────────┐
      ▼           ▼           ▼
  Worker #1   Worker #2   Worker #N
      │           │           │
      └───────────┴───────────┘
                  │
                  ▼
         HTTP POST Webhook URL
                  │
                  ▼
        PostgreSQL (done/failed)
```

---

## 📦 Components

### API Server

Receives webhook jobs through `POST /jobs` and stores them in PostgreSQL.

### Repository

Responsible for persisting and fetching jobs using `FOR UPDATE SKIP LOCKED`, allowing multiple workers to safely consume the queue without race conditions.

### Dispatcher

Periodically polls pending jobs and publishes them into the internal channel consumed by workers.

### Worker Pool

Multiple goroutines process jobs concurrently, improving throughput while controlling resource usage.

### Notifier

Executes the outbound HTTP request to the target webhook endpoint and updates the job status.

---

## 🚀 Running the project

### Docker Compose

```bash
docker compose up --build
```

### Local Development

```bash
# Start PostgreSQL
docker compose up postgres -d

# Copy environment variables
cp .env.example .env

# Run the application
go run ./cmd/api
```

---

## 📡 API

### Create a Job

`POST /jobs`

```bash
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://webhook.site/your-id",
    "body": {
      "event": "user.created",
      "user_id": "123"
    }
  }'
```

Response:

```json
{
  "id": "uuid",
  "status": "pending"
}
```

---

### Health Check

```bash
GET /health
```

```bash
curl http://localhost:8080/health
```

---

## 📊 Job Lifecycle

| Status    | Description               |
| --------- | ------------------------- |
| `pending` | Waiting to be processed   |
| `running` | Currently being processed |
| `done`    | Successfully delivered    |
| `failed`  | Delivery failed           |

Workflow:

```text
pending
    │
    ▼
running
 ┌──┴─────┐
 │        │
 ▼        ▼
done   failed
```

---

## 🧠 Technical Highlights

This project demonstrates several backend engineering concepts:

* Goroutines and channels for concurrent processing
* Worker Pool pattern
* Graceful shutdown with `context.Context` and OS signals
* `sync.WaitGroup` for coordinated worker termination
* `FOR UPDATE SKIP LOCKED` for safe concurrent dequeue
* Persistent job queue using PostgreSQL
* Asynchronous webhook delivery
* Clean project organization with `cmd/` and `internal/`

It serves as a practical example of concurrency, resilience, and scalable backend design in Go.
