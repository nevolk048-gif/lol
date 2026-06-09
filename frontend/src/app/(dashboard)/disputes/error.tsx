"use client";

import { useEffect } from "react";
import { Button } from "@/components/ui/button";
import { AlertTriangle } from "lucide-react";

// Route-level error boundary для /disputes.
// Перехватывает любую ошибку рендера на странице споров и показывает её ТЕКСТ
// прямо в интерфейсе (без необходимости открывать консоль браузера),
// вместо белого экрана "Application error".
export default function DisputesError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    // Дублируем в консоль для тех, у кого есть к ней доступ
    console.error("[disputes] render error:", error);
  }, [error]);

  return (
    <div className="flex flex-col items-center justify-center gap-4 py-16 text-center">
      <AlertTriangle className="h-10 w-10 text-orange-500" />
      <div>
        <h2 className="text-xl font-semibold">Не удалось отобразить споры</h2>
        <p className="text-muted-foreground mt-1">
          Произошла ошибка при загрузке страницы споров. Сама система споров и API
          (<code>GET /api/v1/disputes</code>) продолжают работать.
        </p>
      </div>

      <pre className="max-w-2xl overflow-auto rounded-lg border bg-muted/50 p-4 text-left text-xs text-red-600">
        {error?.message || "Unknown error"}
        {error?.digest ? `\n\ndigest: ${error.digest}` : ""}
      </pre>

      <div className="flex gap-2">
        <Button onClick={() => reset()}>Повторить</Button>
        <Button variant="outline" onClick={() => window.location.reload()}>
          Перезагрузить страницу
        </Button>
      </div>
    </div>
  );
}
