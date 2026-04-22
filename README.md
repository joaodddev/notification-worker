# notification-worker

Sistema de filas de notificações via webhook com worker pool concorrente em Go.

## Arquitetura

```
POST /jobs  →  Postgres (pending)
                    ↓
          Dispatcher (polling 2s)
                    ↓
            channel de jobs
                    ↓
         WorkerPool (5 goroutines)
                    ↓
         HTTP POST no webhook URL
                    ↓
          Postgres (done | failed)
```

### Componentes

- **API HTTP** (`chi`): recebe jobs via `POST /jobs`
- **Repository**: persiste e busca jobs no Postgres com `FOR UPDATE SKIP LOCKED` para evitar race conditions
- **Dispatcher**: faz polling periódico e envia jobs para o channel
- **WorkerPool**: N goroutines consumindo o channel de forma concorrente
- **Notifier**: executa o HTTP POST no URL de destino

## Como rodar

### Com Docker Compose

```bash
docker compose up
```

### Localmente

```bash
# 1. Suba o Postgres
docker compose up postgres -d

# 2. Configure o .env
cp .env.example .env

# 3. Rode
go run ./cmd/api
```

## Endpoints

### `POST /jobs`

Enfileira um job de webhook.

```bash
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://webhook.site/seu-uuid",
    "body": { "event": "user.created", "user_id": "123" }
  }'
```

Resposta:

```json
{
  "id": "uuid-do-job",
  "status": "pending"
}
```

### `GET /health`

```bash
curl http://localhost:8080/health
```

## Status dos jobs

| Status    | Descrição                      |
| --------- | ------------------------------ |
| `pending` | aguardando ser processado      |
| `running` | sendo processado por um worker |
| `done`    | entregue com sucesso           |
| `failed`  | erro na entrega                |

## Conceitos demonstrados

- **Goroutines e channels** para concorrência real
- **`sync.WaitGroup`** para graceful shutdown dos workers
- **`context.Context`** para propagação de cancelamento
- **`FOR UPDATE SKIP LOCKED`** para dequeue seguro com múltiplos workers
- **Graceful shutdown** com `os.Signal`
- **Pool pattern** para limitar concorrência
