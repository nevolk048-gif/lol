import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "GorillaGate — Документация для разработчиков",
  description:
    "Техническая документация GorillaGate: интеграция для мерчантов (казино/гемблинг) и платёжных провайдеров. Создание депозитов, webhook-уведомления, обработка споров (disputes), коды ошибок.",
  keywords: [
    "GorillaGate",
    "payment aggregator",
    "API documentation",
    "merchant integration",
    "provider integration",
    "disputes",
    "webhook",
    "casino payments",
  ],
  robots: { index: true, follow: true },
  openGraph: {
    title: "GorillaGate — Документация для разработчиков",
    description:
      "Интеграция для мерчантов и провайдеров: депозиты, webhooks, споры, коды ошибок.",
    type: "website",
  },
};

// Публичный layout для /docs — НЕ оборачивается в AppShell, поэтому
// страница доступна без авторизации.
export default function DocsLayout({ children }: { children: React.ReactNode }) {
  return <div className="min-h-screen bg-background text-foreground">{children}</div>;
}
