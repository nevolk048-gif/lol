"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { CodeBlock, CodeTabs } from "./code-block";
import {
  BookOpen,
  Rocket,
  CreditCard,
  Activity,
  ShieldAlert,
  AlertCircle,
  Server,
  ArrowRightLeft,
  Webhook,
  KeyRound,
  ExternalLink,
} from "lucide-react";

interface TocItem {
  id: string;
  title: string;
  icon: React.ComponentType<{ className?: string }>;
}

const MERCHANT_TOC: TocItem[] = [
  { id: "m-quickstart", title: "Быстрый старт", icon: Rocket },
  { id: "m-deposit", title: "Создание депозита", icon: CreditCard },
  { id: "m-statuses", title: "Статусы и webhook", icon: Activity },
  { id: "m-disputes", title: "Обработка споров", icon: ShieldAlert },
  { id: "m-errors", title: "Коды ошибок", icon: AlertCircle },
];

const PROVIDER_TOC: TocItem[] = [
  { id: "p-register", title: "Регистрация провайдера", icon: KeyRound },
  { id: "p-incoming", title: "Входящий запрос платежа", icon: ArrowRightLeft },
  { id: "p-response", title: "Требования к ответу", icon: Server },
  { id: "p-webhook", title: "Webhook статусов", icon: Webhook },
  { id: "p-disputes", title: "Споры (disputes)", icon: ShieldAlert },
];

export default function DocsPage() {
  const [active, setActive] = useState<string>("m-quickstart");

  useEffect(() => {
    const ids = [...MERCHANT_TOC, ...PROVIDER_TOC].map((t) => t.id);
    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((e) => e.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top);
        if (visible[0]?.target.id) setActive(visible[0].target.id);
      },
      { rootMargin: "-80px 0px -70% 0px", threshold: 0 }
    );
    ids.forEach((id) => {
      const el = document.getElementById(id);
      if (el) observer.observe(el);
    });
    return () => observer.disconnect();
  }, []);

  return (
    <div className="mx-auto max-w-7xl">
      {/* Top bar */}
      <header className="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-border bg-background/80 px-6 backdrop-blur-xl">
        <Link href="/docs" className="flex items-center gap-2 font-semibold">
          <BookOpen className="h-5 w-5 text-primary" />
          PaymentsGate Docs
        </Link>
        <nav className="flex items-center gap-4 text-sm">
          <Link href="/disputes" className="text-muted-foreground hover:text-foreground">
            Панель споров
          </Link>
          <Link href="/login" className="text-muted-foreground hover:text-foreground">
            Войти
          </Link>
        </nav>
      </header>

      <div className="flex gap-8 px-6 py-8">
        {/* Left: TOC */}
        <aside className="hidden w-64 shrink-0 lg:block">
          <nav className="sticky top-24 space-y-6">
            <TocGroup title="Для мерчантов" items={MERCHANT_TOC} active={active} />
            <TocGroup title="Для провайдеров" items={PROVIDER_TOC} active={active} />
          </nav>
        </aside>

        {/* Right: Content */}
        <main className="min-w-0 flex-1 space-y-16">
          <Intro />

          {/* ---- MERCHANTS ---- */}
          <SectionHeader>Документация для мерчантов (казино / гемблинг)</SectionHeader>
          <MerchantQuickstart />
          <MerchantDeposit />
          <MerchantStatuses />
          <MerchantDisputes />
          <MerchantErrors />

          {/* ---- PROVIDERS ---- */}
          <SectionHeader>Документация для провайдеров (включая MajorPay)</SectionHeader>
          <ProviderRegister />
          <ProviderIncoming />
          <ProviderResponse />
          <ProviderWebhook />
          <ProviderDisputes />

          <Footer />
        </main>
      </div>
    </div>
  );
}

function TocGroup({ title, items, active }: { title: string; items: TocItem[]; active: string }) {
  return (
    <div>
      <p className="mb-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
        {title}
      </p>
      <ul className="space-y-1">
        {items.map((item) => {
          const Icon = item.icon;
          const isActive = active === item.id;
          return (
            <li key={item.id}>
              <a
                href={`#${item.id}`}
                className={`flex items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors ${
                  isActive
                    ? "bg-primary/10 font-medium text-primary"
                    : "text-muted-foreground hover:bg-muted hover:text-foreground"
                }`}
              >
                <Icon className="h-4 w-4 shrink-0" />
                {item.title}
              </a>
            </li>
          );
        })}
      </ul>
    </div>
  );
}

function Section({
  id,
  title,
  children,
}: {
  id: string;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section id={id} className="scroll-mt-24">
      <h2 className="mb-4 border-b border-border pb-2 text-2xl font-bold">{title}</h2>
      <div className="space-y-4 text-sm leading-relaxed text-foreground/90">{children}</div>
    </section>
  );
}

function SectionHeader({ children }: { children: React.ReactNode }) {
  return (
    <h1 className="text-xl font-bold uppercase tracking-wide text-primary">{children}</h1>
  );
}

function P({ children }: { children: React.ReactNode }) {
  return <p className="text-foreground/90">{children}</p>;
}

function Endpoint({ method, path }: { method: string; path: string }) {
  const color =
    method === "GET"
      ? "bg-blue-100 text-blue-800"
      : method === "POST"
        ? "bg-green-100 text-green-800"
        : "bg-amber-100 text-amber-800";
  return (
    <div className="my-3 flex items-center gap-3 rounded-lg border border-border bg-muted/40 px-4 py-2 font-mono text-sm">
      <span className={`rounded px-2 py-0.5 text-xs font-bold ${color}`}>{method}</span>
      <span>{path}</span>
    </div>
  );
}

function Intro() {
  return (
    <div className="rounded-xl border border-border bg-muted/30 p-6">
      <div className="mb-2 flex items-center gap-2">
        <BookOpen className="h-6 w-6 text-primary" />
        <h1 className="text-3xl font-bold">Документация PaymentsGate</h1>
      </div>
      <P>
        <strong>PaymentsGate</strong> — корпоративный агрегатор платежей для маршрутизации
        депозитов казино между 50+ провайдерами. Ниже — техническая документация для двух
        аудиторий: <strong>мерчантов</strong> (казино/гемблинг-операторы) и{" "}
        <strong>провайдеров</strong> (платёжные системы, включая MajorPay).
      </P>
      <div className="mt-4 rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
        Все эндпоинты работают только по <strong>HTTPS</strong>. Базовый префикс API —{" "}
        <code className="rounded bg-amber-100 px-1">/api/v1</code>. Споры доступны через{" "}
        <code className="rounded bg-amber-100 px-1">GET /api/v1/disputes</code> и в UI по пути{" "}
        <code className="rounded bg-amber-100 px-1">/disputes</code>.
      </div>
      <div className="mt-4">
        <p className="mb-1 text-sm font-medium">Формат ответа (конверт):</p>
        <CodeBlock
          lang="json"
          label="response envelope"
          code={`{
  "success": true,
  "data": { },
  "error": null,
  "meta": { "page": 1, "per_page": 20, "total": 134, "total_pages": 7 }
}`}
        />
        <P>
          При <code>success: true</code> полезные данные лежат в <code>data</code>. Если данных
          нет — это <code>null</code> или <code>[]</code>, а не объект-конверт. Не используйте
          паттерн <code>json.data || json</code>.
        </P>
      </div>
    </div>
  );
}

/* ============== MERCHANTS ============== */

function MerchantQuickstart() {
  return (
    <Section id="m-quickstart" title="Быстрый старт">
      <P>
        Чтобы начать принимать депозиты, мерчанту нужен <strong>API-ключ казино</strong>.
      </P>
      <ol className="ml-5 list-decimal space-y-2">
        <li>Зарегистрируйте казино в админ-панели PaymentsGate (или запросите у менеджера).</li>
        <li>
          Получите <code>api_key</code> вида <code>pk_xxx</code>. Он передаётся в каждом запросе
          в заголовке <code>X-API-Key</code>.
        </li>
        <li>
          (Опционально) Укажите <strong>IP whitelist</strong> и <code>webhook_url</code> для
          колбэков.
        </li>
        <li>Создайте первый депозит (см. ниже) — рекомендуется сначала в sandbox-режиме.</li>
      </ol>
      <P>Аутентификация — заголовком:</P>
      <CodeBlock
        lang="http"
        label="auth header"
        code={`X-API-Key: pk_your_casino_api_key
# либо альтернативно:
Authorization: ApiKey pk_your_casino_api_key`}
      />
    </Section>
  );
}

function MerchantDeposit() {
  return (
    <Section id="m-deposit" title="Создание депозита">
      <Endpoint method="POST" path="/api/v1/deposit/create" />
      <P>Параметры тела запроса:</P>
      <div className="overflow-x-auto">
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="border-b border-border text-left">
              <th className="py-2 pr-4 font-semibold">Поле</th>
              <th className="py-2 pr-4 font-semibold">Тип</th>
              <th className="py-2 pr-4 font-semibold">Обяз.</th>
              <th className="py-2 font-semibold">Описание</th>
            </tr>
          </thead>
          <tbody className="font-mono text-xs">
            <Row f="amount" t="number > 0" r="да" d="Сумма депозита" />
            <Row f="currency" t="string(3)" r="да" d="ISO-4217, напр. RUB" />
            <Row f="country" t="string(2)" r="да" d="ISO-3166, напр. RU" />
            <Row f="external_id" t="string" r="нет" d="Ваш ID операции" />
            <Row f="player_id" t="string" r="нет" d="ID игрока" />
            <Row f="merchant_customer_id" t="string" r="нет" d="Привязка плательщика (Payer Affinity)" />
            <Row f="payment_method" t="string" r="нет" d="auto (по умолч.), card, sbp, ..." />
          </tbody>
        </table>
      </div>
      <P>Заголовок <code>Idempotency-Key</code> (опц.) защищает от дублей: повтор с тем же ключом вернёт ту же транзакцию.</P>

      <CodeTabs
        tabs={[
          {
            label: "curl",
            lang: "bash",
            code: `curl -X POST "https://peaceful-hope-production-2262.up.railway.app/api/v1/deposit/create" \\
  -H "X-API-Key: pk_your_casino_api_key" \\
  -H "Content-Type: application/json" \\
  -H "Idempotency-Key: order-10523" \\
  -d '{
    "amount": 5000,
    "currency": "RUB",
    "country": "RU",
    "external_id": "order-10523",
    "player_id": "player-778",
    "payment_method": "auto"
  }'`,
          },
          {
            label: "JavaScript",
            lang: "javascript",
            code: `const res = await fetch(
  "https://peaceful-hope-production-2262.up.railway.app/api/v1/deposit/create",
  {
    method: "POST",
    headers: {
      "X-API-Key": process.env.PG_API_KEY,
      "Content-Type": "application/json",
      "Idempotency-Key": "order-10523",
    },
    body: JSON.stringify({
      amount: 5000,
      currency: "RUB",
      country: "RU",
      external_id: "order-10523",
      player_id: "player-778",
      payment_method: "auto",
    }),
  }
);

const { success, data, error } = await res.json();
if (!success) throw new Error(error.message);

// data.transaction_id, data.status, data.requisite, data.provider
console.log(data.transaction_id, data.requisite?.account_number);`,
          },
          {
            label: "Python",
            lang: "python",
            code: `import requests

resp = requests.post(
    "https://peaceful-hope-production-2262.up.railway.app/api/v1/deposit/create",
    headers={
        "X-API-Key": "pk_your_casino_api_key",
        "Idempotency-Key": "order-10523",
    },
    json={
        "amount": 5000,
        "currency": "RUB",
        "country": "RU",
        "external_id": "order-10523",
        "payment_method": "auto",
    },
    timeout=15,
)

body = resp.json()
if not body["success"]:
    raise RuntimeError(body["error"]["message"])

data = body["data"]
print(data["transaction_id"], data["status"])`,
          },
        ]}
      />

      <P>Ответ <code>201 Created</code>:</P>
      <CodeBlock
        lang="json"
        label="201 Created"
        code={`{
  "success": true,
  "data": {
    "transaction_id": "8f2c3d4e-...-...",
    "status": "WAITING_PAYMENT",
    "requisite": {
      "bank_name": "Sberbank",
      "holder_name": "IVAN I.",
      "account_number": "****1234"
    },
    "provider": { "id": "b0000000-...", "name": "MajorPay" }
  }
}`}
      />

      <P>Проверка статуса транзакции:</P>
      <Endpoint method="GET" path="/api/v1/deposit/status/{transaction_id}" />
    </Section>
  );
}

function Row({ f, t, r, d }: { f: string; t: string; r: string; d: string }) {
  return (
    <tr className="border-b border-border/50">
      <td className="py-1.5 pr-4 text-primary">{f}</td>
      <td className="py-1.5 pr-4 text-muted-foreground">{t}</td>
      <td className="py-1.5 pr-4">{r}</td>
      <td className="py-1.5 font-sans text-foreground/90">{d}</td>
    </tr>
  );
}

function MerchantStatuses() {
  return (
    <Section id="m-statuses" title="Статусы транзакций и webhook-уведомления">
      <P>Жизненный цикл депозитной транзакции:</P>
      <CodeBlock
        lang="http"
        label="lifecycle"
        code={`NEW → ASSIGNED → WAITING_PAYMENT → PAID
                              ↘ EXPIRED
                              ↘ CANCELLED
Выплаты: PAYOUT_SUCCESS / PAYOUT_ERROR`}
      />
      <P>
        PaymentsGate уведомляет мерчанта об изменении статуса POST-запросом на ваш{" "}
        <code>webhook_url</code>. Тело колбэка:
      </P>
      <CodeBlock
        lang="json"
        label="callback body"
        code={`{
  "type": "transaction.updated",
  "object": {
    "transaction_id": "8f2c3d4e-...",
    "external_id": "order-10523",
    "status": "PAID",
    "amount": 5000,
    "currency": "RUB"
  }
}`}
      />
      <ul className="ml-5 list-disc space-y-1">
        <li>Отвечайте <code>200 OK</code> быстро; тяжёлую логику выполняйте асинхронно.</li>
        <li>Делайте обработку идемпотентной — один <code>transaction_id</code> может прийти повторно.</li>
        <li>Критичные операции сверяйте через <code>GET /api/v1/deposit/status/{`{id}`}</code>.</li>
      </ul>
      <P>Пример webhook-обработчика на Node.js (Express) с проверкой подписи:</P>
      <CodeBlock
        lang="javascript"
        label="Node.js webhook handler"
        code={`import express from "express";
import crypto from "crypto";

const app = express();
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
  res.status(200).json({ status: "ok" }); // быстрый ответ

  queueMicrotask(() => {
    if (event.type === "transaction.updated" && event.object.status === "PAID") {
      // идемпотентно зачислить депозит по event.object.external_id
    }
  });
});

app.listen(3000);`}
      />
      <div className="rounded-lg border border-border bg-muted/40 p-3">
        <p className="mb-1 font-medium">Рекомендации по ретраям и таймаутам</p>
        <ul className="ml-5 list-disc space-y-1 text-sm">
          <li>Таймаут запроса: 10–15 секунд.</li>
          <li>Ретраи: экспоненциальная задержка 1s, 2s, 4s, 8s (макс. 4–5 попыток).</li>
          <li>Ретраить только сетевые ошибки и 5xx. Не ретраить 4xx (кроме 429).</li>
          <li>Всегда передавайте <code>Idempotency-Key</code> при создании депозита.</li>
        </ul>
      </div>
    </Section>
  );
}

function MerchantDisputes() {
  return (
    <Section id="m-disputes" title="Обработка споров (disputes)">
      <P>
        Спор (dispute) — это чарджбэк или жалоба по транзакции. Мерчант может создать спор и
        отслеживать его статус. При создании спора трафик провайдера автоматически блокируется до
        разрешения.
      </P>
      <P>Создание спора:</P>
      <Endpoint method="POST" path="/api/v1/disputes" />
      <CodeBlock
        lang="bash"
        label="curl"
        code={`curl -X POST "https://peaceful-hope-production-2262.up.railway.app/api/v1/disputes" \\
  -H "X-API-Key: pk_your_casino_api_key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "transaction_id": "8f2c3d4e-...",
    "reason": "Клиент инициировал чарджбэк: товар не получен"
  }'`}
      />
      <P>Просмотр и фильтрация споров:</P>
      <CodeBlock
        lang="http"
        label="dispute endpoints"
        code={`GET /api/v1/disputes                  # список (фильтры: ?status=NEW&provider_id=&casino_id=&limit=&offset=)
GET /api/v1/disputes/{id}             # один спор
GET /api/v1/disputes/{id}/messages    # переписка по спору
GET /api/v1/disputes/{id}/history     # история изменений
GET /api/v1/disputes/stats            # агрегаты`}
      />
      <P>
        Статусы спора: <code>NEW</code>, <code>UNDER_REVIEW</code>,{" "}
        <code>AWAITING_PROVIDER_RESPONSE</code>, <code>MERCHANT_WON</code>,{" "}
        <code>PROVIDER_WON</code>, <code>CLOSED</code>. Управлять спорами и вести переписку можно в
        UI-панели <Link href="/disputes" className="text-primary underline">/disputes</Link>.
      </P>
    </Section>
  );
}

function MerchantErrors() {
  return (
    <Section id="m-errors" title="Коды ошибок и их значения">
      <div className="overflow-x-auto">
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="border-b border-border text-left">
              <th className="py-2 pr-4 font-semibold">HTTP</th>
              <th className="py-2 pr-4 font-semibold">error.code</th>
              <th className="py-2 pr-4 font-semibold">Значение</th>
              <th className="py-2 font-semibold">Действие клиента</th>
            </tr>
          </thead>
          <tbody>
            <ErrRow h="200" c="—" m="Успех / webhook принят" a="—" />
            <ErrRow h="201" c="—" m="Создано (депозит, спор, сообщение)" a="—" />
            <ErrRow h="400" c="BAD_REQUEST" m="Ошибка валидации, нет полей" a="Исправить запрос, не ретраить" />
            <ErrRow h="401" c="UNAUTHORIZED" m="Нет/неверный ключ или подпись" a="Проверить ключи/подпись" />
            <ErrRow h="403" c="FORBIDDEN" m="Неактивен или IP не в whitelist" a="Проверить статус/whitelist" />
            <ErrRow h="404" c="NOT_FOUND" m="Объект не найден" a="Проверить ID" />
            <ErrRow h="429" c="—" m="Превышен лимит частоты" a="Бэкофф, уважать Retry-After" />
            <ErrRow h="500" c="INTERNAL_ERROR" m="Внутренняя ошибка" a="Ретрай с экспон. задержкой" />
          </tbody>
        </table>
      </div>
    </Section>
  );
}

function ErrRow({ h, c, m, a }: { h: string; c: string; m: string; a: string }) {
  return (
    <tr className="border-b border-border/50">
      <td className="py-1.5 pr-4 font-mono font-semibold">{h}</td>
      <td className="py-1.5 pr-4 font-mono text-xs text-primary">{c}</td>
      <td className="py-1.5 pr-4">{m}</td>
      <td className="py-1.5 text-muted-foreground">{a}</td>
    </tr>
  );
}

/* ============== PROVIDERS ============== */

function ProviderRegister() {
  return (
    <Section id="p-register" title="Регистрация провайдера в системе">
      <P>
        Провайдер (PSP/банк, например <strong>MajorPay</strong>) подключается администратором
        PaymentsGate. При регистрации провайдер получает пару ключей:
      </P>
      <ul className="ml-5 list-disc space-y-1">
        <li><code>api_key</code> (<code>pk_...</code>) — публичный идентификатор.</li>
        <li><code>secret_key</code> (<code>sk_...</code>) — для HMAC-подписи запросов.</li>
        <li><code>base_url</code> — адрес API провайдера, куда PaymentsGate шлёт запросы.</li>
        <li>(Опц.) <strong>IP whitelist</strong> — список разрешённых адресов.</li>
      </ul>
      <P>Защищённые провайдерские запросы аутентифицируются заголовками:</P>
      <CodeBlock
        lang="http"
        label="provider auth"
        code={`X-API-Key: pk_provider_api_key
X-Signature: <hex( HMAC_SHA256(secret_key, raw_request_body) )>`}
      />
      <P>Алгоритм подписи одинаков на обеих сторонах:</P>
      <CodeBlock
        lang="python"
        label="HMAC sign"
        code={`import hmac, hashlib

def sign(raw_body: bytes, secret_key: str) -> str:
    return hmac.new(secret_key.encode(), raw_body, hashlib.sha256).hexdigest()`}
      />
    </Section>
  );
}

function ProviderIncoming() {
  return (
    <Section id="p-incoming" title="Формат входящих запросов на создание платежа">
      <P>
        Когда PaymentsGate маршрутизирует депозит на провайдера, он отправляет POST на{" "}
        <code>{`{base_url}`}/deposit</code> (или согласованный эндпоинт) с заголовками{" "}
        <code>merchant-id</code> и <code>merchant-secret-key</code>. Тело:
      </P>
      <CodeBlock
        lang="json"
        label="incoming payment request"
        code={`{
  "transaction_id": "8f2c3d4e-...",
  "amount": 500000,
  "currency": "RUB",
  "country": "RU",
  "payment_method": "auto"
}`}
      />
      <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
        Суммы передаются в <strong>минорных единицах</strong> (копейки/центы):{" "}
        <code>500000</code> = <code>5000.00 RUB</code>.
      </div>
    </Section>
  );
}

function ProviderResponse() {
  return (
    <Section id="p-response" title="Требования к ответу провайдера (JSON, HTTP-статусы)">
      <P>
        Провайдер должен ответить JSON с реквизитами для оплаты и идентификатором своей
        транзакции (он станет <code>provider_transaction_id</code> в PaymentsGate):
      </P>
      <CodeBlock
        lang="json"
        label="provider response"
        code={`{
  "uuid": "pay_4d0b1f9c-...",
  "status": "waiting",
  "requisite": {
    "bank_name": "Sberbank",
    "holder_name": "IVAN I.",
    "account_number": "40817810099910004312"
  }
}`}
      />
      <div className="overflow-x-auto">
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="border-b border-border text-left">
              <th className="py-2 pr-4 font-semibold">HTTP</th>
              <th className="py-2 font-semibold">Когда возвращать</th>
            </tr>
          </thead>
          <tbody>
            <tr className="border-b border-border/50">
              <td className="py-1.5 pr-4 font-mono font-semibold">200 / 201</td>
              <td className="py-1.5">Платёж принят, реквизиты выданы</td>
            </tr>
            <tr className="border-b border-border/50">
              <td className="py-1.5 pr-4 font-mono font-semibold">400</td>
              <td className="py-1.5">Некорректные данные платежа</td>
            </tr>
            <tr className="border-b border-border/50">
              <td className="py-1.5 pr-4 font-mono font-semibold">401</td>
              <td className="py-1.5">Неверная подпись / merchant-secret-key</td>
            </tr>
            <tr className="border-b border-border/50">
              <td className="py-1.5 pr-4 font-mono font-semibold">409</td>
              <td className="py-1.5">Дубликат transaction_id</td>
            </tr>
            <tr className="border-b border-border/50">
              <td className="py-1.5 pr-4 font-mono font-semibold">422</td>
              <td className="py-1.5">Нет доступных реквизитов для страны/валюты</td>
            </tr>
          </tbody>
        </table>
      </div>
    </Section>
  );
}

function ProviderWebhook() {
  return (
    <Section id="p-webhook" title="Webhook для уведомлений об изменении статуса">
      <P>Провайдер уведомляет PaymentsGate об изменении статуса транзакции:</P>
      <Endpoint method="POST" path="/api/v1/webhook/majorpay" />
      <P>Заголовки:</P>
      <CodeBlock
        lang="http"
        label="webhook headers"
        code={`Content-Type: application/json
X-Major-Timestamp: 1733750400
X-Major-Signature: <hex( HMAC_SHA256(secret_key, raw_body) )>`}
      />
      <P>Тело:</P>
      <CodeBlock
        lang="json"
        label="webhook body"
        code={`{
  "type": "payment.success",
  "object": {
    "uuid": "pay_4d0b1f9c-...",
    "status": "success",
    "amount": 500000,
    "income_amount": 497500
  },
  "secret_key": "sk_..."
}`}
      />
      <P>
        Поле <code>object.uuid</code> — это <code>provider_transaction_id</code>, по которому
        PaymentsGate находит транзакцию. Поддерживаемые <code>type</code>:
      </P>
      <div className="overflow-x-auto">
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="border-b border-border text-left">
              <th className="py-2 pr-4 font-semibold">type</th>
              <th className="py-2 font-semibold">Действие</th>
            </tr>
          </thead>
          <tbody className="font-mono text-xs">
            <tr className="border-b border-border/50"><td className="py-1.5 pr-4">payment.success</td><td className="py-1.5 font-sans">Транзакция → PAID</td></tr>
            <tr className="border-b border-border/50"><td className="py-1.5 pr-4">payment.expired</td><td className="py-1.5 font-sans">Транзакция → EXPIRED</td></tr>
            <tr className="border-b border-border/50"><td className="py-1.5 pr-4">payout.success</td><td className="py-1.5 font-sans">Транзакция → PAYOUT_SUCCESS</td></tr>
            <tr className="border-b border-border/50"><td className="py-1.5 pr-4">payout.error</td><td className="py-1.5 font-sans">Транзакция → PAYOUT_ERROR</td></tr>
          </tbody>
        </table>
      </div>
      <P>
        PaymentsGate всегда возвращает <code>200 OK</code> (<code>{`{"status":"ok"}`}</code>), даже
        если транзакция не найдена — чтобы не провоцировать бесконечные ретраи. Повторяйте отправку
        только при сетевых ошибках/таймауте.
      </P>
    </Section>
  );
}

function ProviderDisputes() {
  return (
    <Section id="p-disputes" title="Споры (disputes) — приём и обработка провайдером">
      <P>
        Когда по транзакции открывается спор, PaymentsGate отправляет провайдеру webhook на{" "}
        <code>{`{base_url}{dispute_endpoint}`}</code> (для MajorPay —{" "}
        <code>https://api.majorpay.io/api/dispute</code>). Путь настраивается на провайдера
        (колонка <code>dispute_endpoint</code>). Провайдер должен принять запрос, ответить{" "}
        <code>200 OK</code> и зарегистрировать спор у себя.
      </P>
      <Endpoint method="POST" path="{base_url}{dispute_endpoint}  →  /api/dispute" />
      <P>Формат данных, который получает провайдер:</P>
      <CodeBlock
        lang="json"
        label="dispute webhook → provider"
        code={`{
  "type": "dispute.created",
  "object": {
    "dispute_id": "d1a2b3c4-...",
    "transaction_id": "8f2c3d4e-...",
    "status": "NEW",
    "reason": "Чарджбэк: товар не получен",
    "amount": 5000,
    "currency": "RUB",
    "created_at": "2026-06-09T12:00:00Z"
  }
}`}
      />
      <P>
        Также провайдер может <strong>инициировать</strong> спор со своей стороны, отправив
        dispute-событие на webhook-эндпоинт PaymentsGate (
        <code>POST /api/v1/webhook/majorpay</code>) с одним из типов:{" "}
        <code>dispute.created</code>, <code>chargeback</code>, <code>chargeback.created</code>,{" "}
        <code>payment.chargeback</code>, <code>payment.dispute</code>:
      </P>
      <CodeBlock
        lang="json"
        label="provider → PaymentsGate (инициировать спор)"
        code={`{
  "type": "dispute.created",
  "object": {
    "uuid": "pay_4d0b1f9c-...",
    "status": "chargeback",
    "amount": 500000
  },
  "secret_key": "sk_..."
}`}
      />
      <P>PaymentsGate при получении dispute-события:</P>
      <ol className="ml-5 list-decimal space-y-1">
        <li>находит транзакцию по <code>object.uuid</code> (<code>provider_transaction_id</code>);</li>
        <li>проверяет, нет ли уже открытого спора (дедупликация);</li>
        <li>создаёт спор в статусе <code>NEW</code>;</li>
        <li>автоматически блокирует трафик провайдера;</li>
        <li>возвращает <code>200 OK</code> с <code>dispute_id</code>.</li>
      </ol>
    </Section>
  );
}

function Footer() {
  return (
    <footer className="mt-16 border-t border-border pt-6 text-sm text-muted-foreground">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <p>© PaymentsGate — Enterprise Payment Aggregator. Все эндпоинты работают по HTTPS.</p>
        <a
          href="#"
          className="inline-flex items-center gap-1 text-primary hover:underline"
        >
          API Reference <ExternalLink className="h-3.5 w-3.5" />
        </a>
      </div>
    </footer>
  );
}
