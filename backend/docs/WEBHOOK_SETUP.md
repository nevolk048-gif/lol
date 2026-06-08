# Webhook Setup Guide

## Локальная разработка - получение публичного URL

### Вариант 1: ngrok (Рекомендуется)

1. Установите ngrok: https://ngrok.com/download

2. Запустите ваше приложение:
```bash
docker-compose up -d
```

3. Создайте туннель к вашему API (порт 8080):
```bash
ngrok http 8080
```

4. ngrok выдаст вам публичный URL, например:
```
Forwarding  https://abc123.ngrok.io -> http://localhost:8080
```

5. Ваш webhook URL будет:
```
https://abc123.ngrok.io/api/webhooks/majorpay
```

### Вариант 2: Cloudflare Tunnel (Бесплатный, без лимитов)

1. Установите cloudflared: https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/

2. Запустите туннель:
```bash
cloudflared tunnel --url http://localhost:8080
```

3. Получите URL вида:
```
https://xyz.trycloudflare.com
```

4. Webhook URL:
```
https://xyz.trycloudflare.com/api/webhooks/majorpay
```

### Вариант 3: localtunnel

1. Установите:
```bash
npm install -g localtunnel
```

2. Запустите:
```bash
lt --port 8080
```

3. Получите URL и используйте его для webhook

---

## Production развертывание

### Вариант 1: Railway.app (Простой деплой)

1. Зарегистрируйтесь на https://railway.app
2. Подключите GitHub репозиторий
3. Railway автоматически развернет ваше приложение
4. Получите URL вида: `https://your-app.up.railway.app`
5. Webhook: `https://your-app.up.railway.app/api/webhooks/majorpay`

### Вариант 2: VPS (Полный контроль)

1. Арендуйте VPS (DigitalOcean, Hetzner, AWS EC2)
2. Настройте домен (например: `api.yourcompany.com`)
3. Установите SSL сертификат (Let's Encrypt)
4. Webhook: `https://api.yourcompany.com/api/webhooks/majorpay`

---

## Настройка webhook URL в базе данных

### Для провайдера (принимать webhook от них):

Провайдер должен отправлять POST запросы на ваш URL.
В таблице `providers` должен быть указан их `secret_key` для проверки подписи.

### Для казино/мерчанта (отправлять webhook им):

Обновите webhook_url в таблице casinos:

```sql
-- Пример для тестового казино
UPDATE casinos 
SET webhook_url = 'https://casino-domain.com/api/webhooks/payments'
WHERE name = 'Test Casino';
```

---

## Тестирование webhook-а

### Отправка тестового webhook локально:

```bash
# Получите secret_key провайдера из БД
PROVIDER_SECRET="sk_ваш_ключ_провайдера"

# Создайте payload
TRANSACTION_ID="pay_test_12345"
TIMESTAMP=$(date +%s)
PAYLOAD='{"type":"payment.success","object":{"uuid":"'$TRANSACTION_ID'","status":"success","amount":150000,"income_amount":120000},"secret_key":"'$PROVIDER_SECRET'"}'

# Сгенерируйте подпись
DATA_TO_SIGN="${TIMESTAMP}.${TRANSACTION_ID}.${PAYLOAD}"
SIGNATURE=$(echo -n "$DATA_TO_SIGN" | openssl dgst -sha256 -hmac "$PROVIDER_SECRET" -hex | cut -d' ' -f2)

# Отправьте webhook
curl -X POST http://localhost:8080/api/webhooks/majorpay \
  -H "Content-Type: application/json" \
  -H "X-Major-Timestamp: $TIMESTAMP" \
  -H "X-Major-Signature: $SIGNATURE" \
  -d "$PAYLOAD"
```

Ответ должен быть:
```json
{"status":"ok"}
```

---

## Проверка логов

После получения webhook проверьте логи:

```sql
-- Последние webhook события
SELECT * FROM audit_logs 
WHERE action IN ('WEBHOOK_RECEIVED', 'WEBHOOK_SENT', 'WEBHOOK_FAILED')
ORDER BY created_at DESC 
LIMIT 20;
```
