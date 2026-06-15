# Effective Mobile Subscriptions

REST-сервис для агрегации данных об онлайн-подписках пользователей.

## Возможности

- CRUDL подписок.
- Подсчет суммарной стоимости подписок за период с фильтрацией по `user_id` и `service_name`.
- PostgreSQL и SQL-миграции.
- JSON-логи через `slog`.
- Конфигурация через переменные окружения или `config.yaml`.
- OpenAPI/Swagger-документация.
- Запуск через Docker Compose.

## Запуск

```bash
cp .env.example .env
# перед запуском замените POSTGRES_PASSWORD и API_KEY на случайные значения
docker compose up --build
```

API будет доступно на `http://localhost:8080`.

Swagger YAML: `http://localhost:8080/swagger/doc.yaml`.

Все бизнес-ручки требуют заголовок `X-API-Key` со значением из `API_KEY`.

## Примеры

Создание подписки:

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

Список:

```bash
curl -H "X-API-Key: $API_KEY" \
  'http://localhost:8080/subscriptions?user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba&limit=20&offset=0'
```

Сумма за период:

```bash
curl -H "X-API-Key: $API_KEY" \
  'http://localhost:8080/subscriptions/total?from=07-2025&to=09-2025&user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba&service_name=Yandex'
```

Период считается включительно по месяцам. Если подписка активна в трех месяцах выбранного периода, ее цена учитывается три раза.
Максимальный период для подсчета суммы — 36 месяцев.

## Локальная разработка

```bash
go test ./...
go run ./cmd/subscriptions
```
