# PaymentsGate

Enterprise-grade payment aggregator SaaS platform. Routes casino deposit requests to payment providers automatically with real-time monitoring, RBAC, and full API integration.

## Architecture

```
Casino → PaymentsGate Aggregator → Provider → Requisite
                ↓
         Routing Engine
    (priority, weight, limits,
     currency, country rules)
```

### Stack

| Layer | Technologies |
|-------|-------------|
| Frontend | React 19, Next.js 15, TypeScript, TailwindCSS, Shadcn-style UI, TanStack Table/Query, Zustand, Framer Motion, Recharts |
| Backend | Go 1.24+, Gin, PostgreSQL, Redis, WebSocket, JWT, Swagger |
| Infrastructure | Docker, Docker Compose, Nginx, GitHub Actions CI |

## Quick Start

### Docker (Recommended)

```bash
docker compose up -d
```

Services:
- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:8080
- **Nginx Proxy**: http://localhost
- **PostgreSQL**: localhost:5432
- **Redis**: localhost:6379

### Default Credentials

| Email | Password | Role |
|-------|----------|------|
| admin@paymentsgate.io | Admin123! | SUPER_ADMIN |
| support@paymentsgate.io | Admin123! | SUPPORT |
| analyst@paymentsgate.io | Admin123! | ANALYST |

> Run `go run ./cmd/seed` to reset admin password hash if login fails.

### Local Development

**Backend:**
```bash
cd backend
cp .env.example .env
go mod download
go run ./cmd/server
```

**Frontend:**
```bash
cd frontend
npm install
npm run dev
```

## Project Structure

```
paymentsgate/
├── frontend/          # Next.js 15 dashboard
│   └── src/
│       ├── app/       # Pages (App Router)
│       ├── components/
│       ├── hooks/
│       ├── services/
│       ├── stores/
│       └── types/
├── backend/
│   ├── cmd/server/    # API entrypoint
│   ├── cmd/seed/      # Database seeder
│   ├── internal/      # Business logic
│   │   ├── auth/
│   │   ├── routing/   # Routing engine
│   │   ├── transactions/
│   │   ├── websocket/
│   │   └── ...
│   ├── pkg/           # Shared packages
│   └── migrations/    # SQL migrations
├── nginx/
├── docker-compose.yml
└── .github/workflows/
```

## API Reference

### Casino API

```bash
# Create deposit
POST /api/v1/deposit/create
Headers: X-API-Key: {casino_api_key}
Body: { "amount": 1000, "currency": "USD", "country": "US" }

# Check status
GET /api/v1/deposit/status/{id}
Headers: X-API-Key: {casino_api_key}
```

### Provider API

```bash
# Get transaction
GET /api/v1/provider/transaction/{id}
Headers: X-API-Key: {provider_api_key}, X-Signature: {hmac_sha256}

# Update status
POST /api/v1/provider/transaction/{id}/status
Body: { "status": "PAID" }
```

### Admin API

All admin endpoints require JWT Bearer token from `POST /api/v1/auth/login`.

Key endpoints:
- `GET /api/v1/dashboard` — Analytics dashboard
- `GET /api/v1/transactions` — Transaction list with filters
- `GET /api/v1/providers` — Provider management
- `GET /api/v1/routing/rules` — Routing configuration
- `POST /api/v1/sandbox/setup` — Initialize sandbox environment

### WebSocket

```
ws://localhost:8080/ws?token={jwt_access_token}
```

Events: `transaction.new`, `transaction.status`, `error`, `provider.connected`, `monitoring.update`

## Routing Engine

Automatic routing on new deposit:

1. Find active providers matching route rules
2. Check requisites (status, limits, currency, country)
3. Apply priority and weighted selection
4. Reserve requisite daily limit
5. Assign provider + requisite
6. Broadcast WebSocket event
7. Write audit log

## Roles (RBAC)

| Role | Permissions |
|------|------------|
| SUPER_ADMIN | Full access, user management |
| ADMIN | Providers, casinos, routing, sandbox |
| SUPPORT | View transactions, logs |
| ANALYST | Dashboard, analytics, reports |

## Security

- JWT Access + Refresh tokens
- HMAC SHA256 request signing
- IP Whitelist for API keys
- Rate limiting (100 RPS default)
- AES-256 encryption for sensitive data
- Full audit logging

## Sandbox Mode

All sandbox entities are tagged with `is_sandbox: true`. Use the Sandbox page or API:

```bash
POST /api/v1/sandbox/setup
POST /api/v1/sandbox/deposit
POST /api/v1/sandbox/generate-traffic
POST /api/v1/sandbox/generate-stats
```

## License

Proprietary — PaymentsGate Enterprise Platform
