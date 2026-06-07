import { create } from "zustand";
import { persist } from "zustand/middleware";
import en from "@/locales/en.json";
import ru from "@/locales/ru.json";

type Locale = "en" | "ru";

const translations = { en, ru };

interface I18nState {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: string) => string;
}

export const useI18n = create<I18nState>()(
  persist(
    (set, get) => ({
      locale: "ru",
      setLocale: (locale) => set({ locale }),
      t: (key: string) => {
        const { locale } = get();
        return (translations[locale] as any)[key] || key;
      },
    }),
    {
      name: "i18n-storage",
    }
  )
);
