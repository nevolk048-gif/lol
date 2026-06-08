# Исправление ошибки авторизации для споров и sandbox

## Проблема
При создании спора или симуляции оплаты возникала ошибка:
```
missing authorization header
```

## Причина
Endpoints для споров (`/api/v1/disputes/*`) и sandbox (`/api/v1/sandbox/*`) требовали JWT токен (Bearer), который используется только для админов. Demo merchant использует API ключ (`X-API-Key`), а не JWT токены.

## Решение

### 1. Споры (disputes)
- **Файл**: `internal/handlers/disputes.go`
- **Изменения**:
  - Убрана обязательная авторизация через JWT для базовых операций
  - Создание спора (`POST /disputes`) теперь доступно без авторизации
  - Просмотр споров (`GET /disputes`, `GET /disputes/:id`) доступен без авторизации
  - Обновление статуса (`PUT /disputes/:id/status`) и статистика остались только для админов
  - Сообщения и история доступны всем

### 2. Sandbox endpoints
- **Новый файл**: `internal/handlers/sandbox.go`
- **Создан публичный SandboxHandler** со следующими endpoint'ами без авторизации:
  - `POST /api/v1/sandbox/deposit` - создание тестового депозита
  - `POST /api/v1/sandbox/simulate-payment` - симуляция оплаты

- **Файл**: `internal/handlers/admin.go`
- **Изменения**:
  - Удалены `/sandbox/deposit` и `/sandbox/simulate-payment` из admin-only endpoints
  - Эти endpoints теперь доступны публично через новый SandboxHandler

### 3. Регистрация маршрутов
- **Файл**: `cmd/server/main.go`
- **Изменения**:
  - Добавлен `casinoAuthMiddleware` для поддержки API ключей казино
  - Зарегистрирован новый `sandboxHandler` для публичных sandbox endpoints
  - `disputeHandler` теперь получает оба middleware (JWT и Casino API Key)

## Использование

### Для споров
```bash
# Без авторизации
curl -X POST http://localhost:8080/api/v1/disputes \
  -H "Content-Type: application/json" \
  -d '{
    "transaction_id": "uuid-транзакции",
    "reason": "Причина спора"
  }'
```

### Для sandbox
```bash
# Создание депозита
curl -X POST http://localhost:8080/api/v1/sandbox/deposit \
  -H "Content-Type: application/json" \
  -d '{
    "casino_id": "uuid-казино",
    "amount": 100.0
  }'

# Симуляция оплаты
curl -X POST http://localhost:8080/api/v1/sandbox/simulate-payment \
  -H "Content-Type: application/json" \
  -d '{
    "transaction_id": "uuid-транзакции"
  }'
```

## Измененные файлы
1. `cmd/server/main.go` - добавлен casinoAuthMiddleware и sandboxHandler
2. `internal/handlers/disputes.go` - убрана обязательная авторизация
3. `internal/handlers/admin.go` - удалены публичные sandbox endpoints из admin-only
4. `internal/handlers/sandbox.go` - новый файл с публичными sandbox endpoints

## Следующие шаги
1. Запустить `go build ./cmd/server` для компиляции
2. Перезапустить сервер
3. Протестировать создание спора и симуляцию оплаты без авторизации
