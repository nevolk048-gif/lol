# Webhook Documentation

## Входящие Webhook-и (от провайдера к нам)

### Endpoint
```
POST /api/webhooks/majorpay
```

### События
- `payment.success` - успешная оплата
- `payment.expired` - истек срок оплаты
- `payout.success` - успешная выплата
- `payout.error` - ошибка выплаты

### Пример Payload
```json
{
  "type": "payment.success",
  "object": {
    "uuid": "pay_4d0b1f...",
    "status": "success",
    "amount": 150000,
    "income_amount": 120000
  },
  "secret_key": "sk_2d1c7f5807e2428a880e810be7278d8f"
}
```

### Проверка подписи (HMAC-SHA256)
**Headers:**
- `X-Major-Timestamp` - Unix timestamp
- `X-Major-Signature` - HMAC-SHA256 подпись

**Формула подписи:**
```
HMAC-SHA256(timestamp + "." + trade_id + "." + raw_body, provider_secret_key)
```

**Где:**
- `timestamp` - значение из заголовка X-Major-Timestamp
- `trade_id` - object.uuid из тела запроса
- `raw_body` - исходное тело запроса (до парсинга JSON)
- `provider_secret_key` - секретный ключ провайдера из БД

### Ответ
Всегда возвращаем `200 OK`:
```json
{"status": "ok"}
```

---

## Исходящие Webhook-и (от нас к мерчанту/казино)

### Когда отправляется
Автоматически при смене статуса транзакции на финальный:
- `PAID` → `payment.success`
- `EXPIRED` → `payment.expired`
- `PAYOUT_SUCCESS` → `payout.success`
- `PAYOUT_ERROR` → `payout.error`
- `CANCELLED` → `payment.cancelled`

### Webhook URL
Берется из поля `webhook_url` в таблице `casinos`.

### Пример Payload
```json
{
  "type": "payment.success",
  "object": {
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "status": "PAID",
    "amount": 1500.00,
    "income_amount": 1500.00,
    "external_id": "casino_tx_123"
  },
  "secret_key": "sk_casino_secret_key"
}
```

### Генерация подписи (HMAC-SHA256)
**Headers:**
- `X-Major-Timestamp` - Unix timestamp
- `X-Major-Signature` - HMAC-SHA256 подпись
- `Content-Type: application/json`

**Формула подписи:**
```
HMAC-SHA256(timestamp + "." + transaction_id + "." + json_body, casino_secret_key)
```

### Проверка на стороне мерчанта (PHP пример)
```php
$timestamp = $_SERVER["HTTP_X_MAJOR_TIMESTAMP"] ?? "";
$signature = $_SERVER["HTTP_X_MAJOR_SIGNATURE"] ?? "";

// Read raw body
$rawBody = file_get_contents("php://input");
$payload = json_decode($rawBody, true);

$transactionId = $payload["object"]["uuid"] ?? "";
$secretKey = "sk_ваш_секретный_ключ"; // Из настроек казино

// Generate expected signature
$dataToSign = $timestamp . "." . $transactionId . "." . $rawBody;
$expected = hash_hmac("sha256", $dataToSign, $secretKey);

// Verify signature
if (!hash_equals($expected, $signature)) {
  http_response_code(401);
  exit("Invalid Signature");
}

// Process webhook
// ...

// MUST return 200 OK
http_response_code(200);
echo "OK";
```

### Логирование
Все попытки отправки webhook-ов логируются в таблицу `audit_logs`:
- `WEBHOOK_SENT` - успешная отправка
- `WEBHOOK_FAILED` - ошибка отправки

---

## Настройка

### Для провайдера
1. В таблице `providers` должен быть заполнен `secret_key`
2. Провайдер должен отправлять webhook-и на наш endpoint

### Для казино (мерчанта)
1. В таблице `casinos` заполнить:
   - `webhook_url` - URL endpoint для получения колбэков
   - `secret_key` - будет использоваться для подписи наших webhook-ов
2. На стороне казино реализовать endpoint, который:
   - Проверяет подпись `X-Major-Signature`
   - Обрабатывает событие
   - Возвращает `200 OK`

### Генерация secret_key для существующих казино
Миграция `009_add_casino_secret_key.sql` автоматически генерирует случайные ключи для всех существующих записей.

Для новых казино ключ должен генерироваться при создании записи.
