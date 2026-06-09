"use client";

import { useState } from "react";
import { Check, Copy } from "lucide-react";

type Lang = "bash" | "json" | "javascript" | "python" | "http";

export function CodeBlock({
  code,
  lang = "bash",
  label,
}: {
  code: string;
  lang?: Lang;
  label?: string;
}) {
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // clipboard может быть недоступен (http/iframe) — тихо игнорируем
    }
  };

  return (
    <div className="group relative my-4 overflow-hidden rounded-lg border border-border bg-zinc-950">
      <div className="flex items-center justify-between border-b border-zinc-800 px-4 py-2">
        <span className="text-xs font-medium uppercase tracking-wide text-zinc-400">
          {label ?? lang}
        </span>
        <button
          onClick={copy}
          className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100"
          aria-label="Скопировать код"
        >
          {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
          {copied ? "Скопировано" : "Копировать"}
        </button>
      </div>
      <pre className="overflow-x-auto p-4 text-sm leading-relaxed">
        <code className="font-mono text-zinc-100">{code}</code>
      </pre>
    </div>
  );
}

// Набор вкладок с примерами на разных языках (curl / JS / Python).
export function CodeTabs({
  tabs,
}: {
  tabs: { label: string; lang: Lang; code: string }[];
}) {
  const [active, setActive] = useState(0);

  return (
    <div className="my-4">
      <div className="flex gap-1 border-b border-border">
        {tabs.map((t, i) => (
          <button
            key={t.label}
            onClick={() => setActive(i)}
            className={`px-3 py-1.5 text-sm font-medium transition-colors ${
              i === active
                ? "border-b-2 border-primary text-foreground"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>
      <CodeBlock code={tabs[active].code} lang={tabs[active].lang} label={tabs[active].label} />
    </div>
  );
}
