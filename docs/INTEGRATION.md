# PaymentsGate — Документация по интеграции

> **PaymentsGate** — корпоративный агрегатор платежей для маршрутизации депозитов
> казино между 50+ платёжными провайдерами. Документ описывает интеграцию для
> **платёжных провайдеров** (которые принимают трафик от PaymentsGate) и для
> **мерчантов** (казино и гемблинг-операторов, которые отправляют депозиты).

---

## 0. Общие сведения

| Параметр | Значение |
|----------|----------|
| Протокол | **Только HTTPS** (HTTP-запросы отклоняются/редиректятся) |
| Base URL | `https://<your-paymentsgate-host>` (прод-хост на Railway; замените на ваш) |
| Префикс API | `/api/v1` |
| Формат | `application/json` (UTF-8) |
| Аутентификация мерчанта | заголовок `X-API-Key` |
| Аутентификация провайдера | заголовок `X-API-Key` + подпись `X-Signature` |
| UI-панель споров | `https://<frontend-host>/disputes` |

### Формат ответа (конверт)

Все ответы обёрнуты в единый конверт:

```json
{
  "success": true,
  "data": { },
  "error": null,
  "meta": { "page": 1, "per_page": 20, "total": 134, "total_pages": 7 }
}
```

При ошибке:

```json
{
  "success": false,
  "error": { "code": "BAD_REQUEST", "message": "amount is required" }
}
```

> ⚠️ **Важно для клиентов:** при `success: true` полезные данные лежат в поле
> `data`. Если `data` отсутствует/пустой — это `null` или `[]`, а **не** весь
> объект-конверт. Не используйте паттерн `json.data || json`.

---

# ЧАСТЬ 1. Для платёжных провайдеров

Провайдер — это PSP/банк, на который PaymentsGate отправляет депозитный трафик и
от которого получает уведомления о смене статуса транзакции.

## 1.1. Аутентификация и безопасность

При обращении к защищённым провайдерским эндпоинтам PaymentsGate:

| Заголовок | Назначение |
|-----------|------------|
| `X-API-Key` | Публичный ключ провайдера (`pk_...`) |
| `X-Signature` | HMAC-SHA256 подпись **сырого тела запроса** секретным ключом (`sk_...`), в **hex** |

**Алгоритм подписи (идентичен на обеих сторонах):**

```
signature = HEX( HMAC_SHA256( secret_key, raw_request_body ) )
```

Пример на Python:

```python
import hmac, hashlib

def sign(raw_body: bytes, secret_key: str) -> str:
    return hmac.new(secret_key.encode(), raw_body, hashlib.sha256).hexdigest()
```

**IP whitelist.** Для провайдера можно задать список разрешённых IP. Если список
непустой и IP источника в него не входит — запрос отклоняется (`403 FORBIDDEN`,
`IP not whitelisted`). Пустой список = разрешены все адреса.

## 1.2. Уведомления о статусе транзакции (Webhook → PaymentsGate)

Провайдер уведомляет PaymentsGate о смене статуса, отправляя POST на webhook-эндпоинт.

```
POST /api/v1/webhook/majorpay
```

**Заголовки:**

| Заголовок | Обязателен | Описание |
|-----------|:---:|----------|
| `Content-Type: application/json` | да | |
| `X-Major-Timestamp` | да | Unix-время отправки |
| `X-Major-Signature` | да | HMAC-SHA256 (hex) от тела запроса |

**Тело запроса:**

```json
{
  "type": "payment.success",
  "object": {
    "uuid": "pay_4d0b1f9c-...",
    "status": "success",
    "amount": 500000,
    "income_amount": 497500
  },
  "secret_key": "sk_..."
}
```

> Денежные суммы — в **минорных единицах** (копейки/центы). `500000` = `5000.00 RUB`.
> Поле `object.uuid` — это `provider_transaction_id`, по которому PaymentsGate
> находит свою транзакцию.

**Поддерживаемые `type`:**

| `type` | Действие в PaymentsGate |
|--------|--------------------------|
| `payment.success` | Транзакция → `PAID` |
| `payment.expired` | Транзакция → `EXPIRED` |
| `payout.success` | Транзакция → `PAYOUT_SUCCESS` |
| `payout.error` | Транзакция → `PAYOUT_ERROR` |
| `dispute.created`, `chargeback`, `chargeback.created`, `payment.chargeback`, `payment.dispute` | Открывается спор + автоблокировка трафика провайдера |

**Ответ.** PaymentsGate всегда возвращает `200 OK` (`{"status":"ok"}`), даже если
транзакция не найдена — чтобы не провоцировать бесконечные ретраи. Повторяйте
отправку только при сетевых ошибках/таймауте.

## 1.3. Споры и чарджбэки (формат данных)

Чтобы инициировать спор со стороны провайдера, отправьте webhook с dispute-типом:

```json
{
  "type": "dispute.created",
  "object": {
    "uuid": "pay_4d0b1f9c-...",
    "status": "chargeback",
    "amount": 500000
  },
  "secret_key": "sk_..."
}
```

Поведение PaymentsGate при получении dispute-события:

1. Находит транзакцию по `object.uuid` (`provider_transaction_id`).
2. Проверяет, нет ли уже **открытого** спора (дедупликация).
3. Создаёт спор в статусе `NEW`.
4. **Автоматически блокирует трафик** провайдера до разрешения спора.
5. Возвращает `200 OK` с `dispute_id`.

## 1.4. Коды ошибок (провайдерская сторона)

| HTTP | `error.code` | Значение |
|-----:|--------------|----------|
| 400 | `BAD_REQUEST` | Некорректный JSON / отсутствуют поля |
| 401 | `UNAUTHORIZED` | Нет/неверный `X-API-Key`, отсутствуют заголовки подписи, неверная подпись |
| 403 | `FORBIDDEN` | Провайдер неактивен или IP не в whitelist |
| 404 | `NOT_FOUND` | Объект не найден |
| 500 | `INTERNAL_ERROR` | Внутренняя ошибка |

---

# ЧАСТЬ 2. Для мерчантов (казино / гемблинг-операторы)

Мерчант создаёт депозитные запросы, а PaymentsGate подбирает провайдера и реквизиты.

## 2.1. Аутентификация

| Заголовок | Описание |
|-----------|----------|
| `X-API-Key` | API-ключ казино. Альтернатива: `Authorization: ApiKey <key>` |
| `Idempotency-Key` | *(опц.)* Защита от дублей — повтор с тем же ключом вернёт ту же транзакцию |

> Для казино тоже поддерживается **IP whitelist** (как у провайдеров).

## 2.2. Создание депозитного запроса

```
POST /api/v1/deposit/create
```

**Тело:**

| Поле | Тип | Обяз. | Описание |
|------|-----|:---:|----------|
| `amount` | number > 0 | да | Сумма депозита |
| `currency` | string(3) | да | ISO-4217, напр. `RUB` |
| `country` | string(2) | да | ISO-3166, напр. `RU` |
| `external_id` | string | нет | Ваш ID операции |
| `player_id` | string | нет | ID игрока |
| `merchant_customer_id` | string | нет | Для привязки плательщика (Payer Affinity) |
| `payment_method` | string | нет | `auto` (по умолч.), `card`, `sbp`, ... |

### Пример: curl

```bash
curl -X POST "https://<your-paymentsgate-host>/api/v1/deposit/create" \
  -H "X-API-Key: $PG_API_KEY" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: order-10523" \
  -d '{
    "amount": 5000,
    "currency": "RUB",
    "country": "RU",
    "external_id": "order-10523",
    "player_id": "player-778",
    "payment_method": "auto"
  }'
```

### Пример: JavaScript (fetch)

```javascript
const res = await fetch("https://<your-paymentsgate-host>/api/v1/deposit/create", {
  method: "POST",
  headers: {
    "X-API-Key": process.env.PG_API_KEY,
    "Content-Type": "application/json",
    "Idempotency-Key": "order-10523",
  },
  body: JSON.stringify({
    amount: 5000, currency: "RUB", country: "RU",
    external_id: "order-10523", player_id: "player-778", payment_method: "auto",
  }),
});

const { success, data, error } = await res.json();
if (!success) throw new Error(error.message);
// data.transaction_id, data.status, data.requisite, data.provider
console.log(data.transaction_id, data.requisite?.account_number);
```

### Пример: Python (requests)

```python
import requests

resp = requests.post(
    "https://<your-paymentsgate-host>/api/v1/deposit/create",
    headers={"X-API-Key": PG_API_KEY, "Idempotency-Key": "order-10523"},
    json={"amount": 5000, "currency": "RUB", "country": "RU",
          "external_id": "order-10523", "payment_method": "auto"},
    timeout=15,
)
body = resp.json()
if not body["success"]:
    raise RuntimeError(body["error"]["message"])
data = body["data"]
print(data["transaction_id"], data["status"])
```

**Ответ (`201 Created`):**

```json
{
  "success": true,
  "data": {
    "transaction_id": "8f2c...-...",
    "status": "WAITING_PAYMENT",
    "requisite": { "bank_name": "Sberbank", "holder_name": "...", "account_number": "****1234" },
    "provider": { "id": "b000...", "name": "MajorPay" }
  }
}
```

### Проверка статуса

```
GET /api/v1/deposit/status/{transaction_id}
```

**Статусы транзакции:** `NEW` → `ASSIGNED` → `WAITING_PAYMENT` → `PAID`
(или `EXPIRED` / `CANCELLED`); для выплат — `PAYOUT_SUCCESS` / `PAYOUT_ERROR`.

## 2.3. Обработка колбэков от PaymentsGate

PaymentsGate уведомляет мерчанта об изменении статуса на ваш `webhook_url`.
Тело колбэка:

```json
{
  "type": "transaction.updated",
  "object": {
    "transaction_id": "8f2c...",
    "external_id": "order-10523",
    "status": "PAID",
    "amount": 5000,
    "currency": "RUB"
  }
}
```

**Рекомендации:**

- Отвечайте `200 OK` **быстро** (тяжёлую логику выполняйте асинхронно).
- Делайте обработку **идемпотентной** (один `transaction_id` может прийти повторно).
- Доверяйте статусу из колбэка, но критичные операции сверяйте через
  `GET /deposit/status/{id}`.

### Пример webhook-обработчика на Node.js (Express)

```javascript
import express from "express";
import crypto from "crypto";

const app = express();
// нужен сырой body для проверки подписи
app.use("/webhooks/paymentsgate", express.raw({ type: "application/json" }));

const SECRET = process.env.PG_WEBHOOK_SECRET;

app.post("/webhooks/paymentsgate", (req, res) => {
  const raw = req.body; // Buffer
  const signature = req.get("X-Signature") || "";

  // Подпись: HEX(HMAC_SHA256(secret, raw_body))
  const expected = crypto.createHmac("sha256", SECRET).update(raw).digest("hex");
  const ok =
    signature.length === expected.length &&
    crypto.timingSafeEqual(Buffer.from(signature), Buffer.from(expected));

  if (!ok) return res.status(401).json({ error: "invalid signature" });

  const event = JSON.parse(raw.toString("utf8"));

  // Быстрый ответ; обработку — в фон
  res.status(200).json({ status: "ok" });

  queueMicrotask(() => {
    if (event.type === "transaction.updated" && event.object.status === "PAID") {
      // идемпотентно зачислить депозит по event.object.external_id
    }
  });
});

app.listen(3000);
```

## 2.4. Просмотр споров

### В UI-панели

Откройте `https://<frontend-host>/disputes`. Доступны: список споров с фильтром по
статусу, переписка по спору и смена статуса. Индикатор спора также виден в карточке
транзакции на `/transactions`.

### Через API

```
GET /api/v1/disputes                  # список (фильтры: ?status=NEW&provider_id=&casino_id=&limit=&offset=)
GET /api/v1/disputes/{id}             # один спор
GET /api/v1/disputes/{id}/messages    # переписка
GET /api/v1/disputes/{id}/history     # история изменений
GET /api/v1/disputes/stats            # агрегаты
```

```bash
curl "https://<your-paymentsgate-host>/api/v1/disputes?status=NEW&limit=50" \
  -H "Authorization: Bearer $JWT"
```

**Статусы спора:** `NEW`, `UNDER_REVIEW`, `AWAITING_PROVIDER_RESPONSE`,
`MERCHANT_WON`, `PROVIDER_WON`, `CLOSED`.

> ✅ Эндпоинт `GET /api/v1/disputes` исправен и доступен по HTTPS. Если UI-страница
> `/disputes` отображает ошибку — это проблема фронтенда, API при этом работает.

## 2.5. Рекомендации по ретраям и таймаутам

| Параметр | Рекомендация |
|----------|--------------|
| Таймаут запроса | 10–15 секунд |
| Стратегия ретраев | Экспоненциальная задержка: 1s, 2s, 4s, 8s (макс. 4–5 попыток) |
| Что ретраить | Сетевые ошибки и `5xx`. **Не** ретраить `4xx` (кроме `429`) |
| Идемпотентность | Всегда передавайте `Idempotency-Key` на создание депозита |
| `429 Too Many Requests` | Снизьте RPS, уважайте `Retry-After` |

---

## Частые ошибки при интеграции

1. **Потерян префикс `/api/v1`.** Все эндпоинты живут под `/api/v1`. Базовый URL —
   это голый хост; путь добавляется отдельно (`<host>/api/v1/disputes`, а не
   `<host>/disputes`).
2. **Распаковка конверта через `data || json`.** При пустом `data` так наружу уходит
   весь объект-конверт, и `list.map(...)` падает. Берите строго `response.data`
   (это `null`/`[]`, если данных нет).
3. **Подпись от изменённого тела.** HMAC считается от **сырых байт** тела. Не
   пересериализуйте JSON перед проверкой подписи — порядок ключей/пробелы изменят хеш.
4. **Суммы в неверных единицах.** Webhook'и провайдера используют **минорные**
   единицы (копейки). `500000` ≠ `500000.00`.
5. **Блокирующая обработка колбэка.** Долгая логика до ответа `200` ведёт к таймаутам
   и ретраям. Отвечайте сразу, обрабатывайте в фоне.
6. **Неидемпотентная обработка.** Колбэки и webhook'и могут приходить повторно —
   защищайтесь по `transaction_id` / `external_id`.
7. **HTTP вместо HTTPS.** Все интеграции — только по HTTPS.
8. **Игнорирование IP whitelist.** Если задан whitelist, запросы с других IP получают
   `403 FORBIDDEN`.

## Сводная таблица HTTP-статусов

| HTTP | Когда | Действие клиента |
|-----:|-------|------------------|
| 200 | Успех / webhook принят | — |
| 201 | Создано (депозит, спор, сообщение) | — |
| 400 | Ошибка валидации | Исправить запрос, **не** ретраить |
| 401 | Проблема аутентификации/подписи | Проверить ключи/подпись |
| 403 | Неактивен или IP не разрешён | Проверить статус/whitelist |
| 404 | Не найдено | Проверить ID |
| 429 | Лимит частоты | Бэкофф, уважать `Retry-After` |
| 500 | Внутренняя ошибка | Ретрай с экспоненциальной задержкой |
