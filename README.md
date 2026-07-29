# Subscriptions API

A REST service for aggregating user subscription data.

## Features

- CRUDL for subscriptions.
- Cost roll-up over a date range, filterable by `user_id` and `service_name`.
- PostgreSQL with SQL migrations.
- JSON logging via `slog`.
- Configuration via environment variables or `config.yaml`.
- OpenAPI/Swagger docs.
- Runs via Docker Compose.

## Running it

```bash
cp .env.example .env
# replace POSTGRES_PASSWORD and API_KEY with random values before starting
docker compose up --build
```

The API is available on `http://localhost:8080`.

Swagger YAML: `http://localhost:8080/swagger/doc.yaml`.

Every business endpoint requires an `X-API-Key` header matching `API_KEY`.

## Examples

Create a subscription:

```bash
curl -X POST http://localhost:8080/subscriptions \
  -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{
    "service_name": "Yandex Plus",
    "price": 400,
    "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
    "start_date": "07-2025"
  }'
```

List:

```bash
curl -H "X-API-Key: $API_KEY" \
  'http://localhost:8080/subscriptions?user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba&limit=20&offset=0'
```

Total cost over a range:

```bash
curl -H "X-API-Key: $API_KEY" \
  'http://localhost:8080/subscriptions/total?from=07-2025&to=09-2025&user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba&service_name=Yandex'
```

The range is inclusive by month — a subscription active across three months of the selected
period is counted three times. The maximum range for a total is 36 months.

## Local development

```bash
go test ./...
go run ./cmd/subscriptions
```
