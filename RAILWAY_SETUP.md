# Railway Deployment Guide

## Настройка Backend на Railway

### Шаг 1: Создайте новый проект в Railway

1. Зайдите на https://railway.app
2. Нажмите "New Project"
3. Выберите "Deploy from GitHub repo"
4. Выберите репозиторий `lol` (или как он у вас называется)

### Шаг 2: Настройте переменные окружения

В Railway Dashboard для сервиса backend добавьте следующие переменные:

```
DATABASE_URL=postgresql://username:password@host:port/database
REDIS_URL=redis://host:port
JWT_SECRET=your-secret-key-here
PORT=8080
GIN_MODE=release
```

### Шаг 3: Добавьте PostgreSQL и Redis

1. В проекте нажмите "New" → "Database" → "Add PostgreSQL"
2. Railway автоматически создаст переменную `DATABASE_URL`
3. Повторите для Redis: "New" → "Database" → "Add Redis"
4. Railway создаст `REDIS_URL`

### Шаг 4: Настройка Root Directory (если нужно)

Если Railway не видит backend:

1. В настройках сервиса перейдите в "Settings"
2. Найдите "Root Directory"
3. Установите: `backend`
4. Сохраните

### Шаг 5: Выполните миграции

После первого деплоя выполните миграции через Railway CLI или создайте отдельный сервис для миграций:

```bash
# Локально с Railway CLI
railway run --service backend bash -c "cd backend && go run run_migration_006.go"
```

Или добавьте pre-deploy скрипт в railway.toml.

### Шаг 6: Получите URL вашего API

1. В Dashboard найдите ваш backend сервис
2. Перейдите в "Settings" → "Domains"
3. Railway автоматически создаст домен типа: `your-app.up.railway.app`
4. Или добавьте свой кастомный домен

### Ваш Webhook URL:

```
https://your-backend-app.up.railway.app/api/webhooks/majorpay
```

## Настройка Frontend (опционально)

Если хотите развернуть frontend отдельно:

1. Создайте новый сервис в том же проекте
2. В Root Directory установите: `frontend`
3. Добавьте переменные окружения:
   ```
   NEXT_PUBLIC_API_URL=https://your-backend-app.up.railway.app
   NEXT_PUBLIC_WS_URL=wss://your-backend-app.up.railway.app
   ```

## Проверка работы

После деплоя проверьте:

```bash
# Health check
curl https://your-backend-app.up.railway.app/api/health

# Webhook endpoint
curl -X POST https://your-backend-app.up.railway.app/api/webhooks/majorpay \
  -H "Content-Type: application/json" \
  -H "X-Major-Timestamp: 1234567890" \
  -H "X-Major-Signature: test" \
  -d '{"type":"payment.success","object":{"uuid":"test"},"secret_key":"test"}'
```

## Troubleshooting

### Backend не запускается

Проверьте логи в Railway Dashboard:
- Нажмите на сервис backend
- Перейдите на вкладку "Deployments"
- Кликните на последний деплой
- Изучите логи сборки и запуска

### 404 ошибка

Убедитесь что:
1. Root Directory установлен правильно (для monorepo)
2. PORT переменная окружения установлена
3. Backend успешно собрался (проверьте Build Logs)

### Database connection failed

Проверьте:
1. DATABASE_URL правильно установлен
2. PostgreSQL сервис запущен в Railway
3. Миграции выполнены

## Альтернатива: Разделенный деплой

Если хотите развернуть только backend:

1. Создайте отдельный репозиторий только с папкой backend
2. Или используйте Git subtree:
   ```bash
   git subtree push --prefix backend origin backend-only
   ```
3. Задеплойте эту ветку на Railway

## Webhook Testing

После деплоя используйте ваш Railway URL для тестирования webhook-ов согласно документации в `backend/docs/WEBHOOK_SETUP.md`
