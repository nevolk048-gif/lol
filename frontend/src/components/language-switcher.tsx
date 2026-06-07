"use client";

import { useI18n } from "@/hooks/use-i18n";
import { Globe } from "lucide-react";

export function LanguageSwitcher() {
  const { locale, setLocale } = useI18n();

  return (
    <button
      onClick={() => setLocale(locale === "en" ? "ru" : "en")}
      className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm hover:bg-accent transition-colors"
      title="Change language"
    >
      <Globe className="h-4 w-4" />
      <span className="font-medium">{locale === "en" ? "RU" : "EN"}</span>
    </button>
  );
}
