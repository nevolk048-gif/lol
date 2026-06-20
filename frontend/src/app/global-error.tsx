"use client";

import { useEffect } from "react";

// Next.js 15 global error boundary — перехватывает ошибки в корневом layout.
// Показывает текст ошибки прямо в интерфейсе вместо "Application error".
export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error("[global-error]", error);
  }, [error]);

  return (
    <html lang="en">
      <body style={{ fontFamily: "monospace", padding: "2rem", background: "#0a0a0a", color: "#f5f5f5" }}>
        <div style={{ maxWidth: 640, margin: "0 auto" }}>
          <h1 style={{ color: "#ef4444", marginBottom: "0.5rem" }}>Client-side exception</h1>
          <pre
            style={{
              background: "#1a1a1a",
              border: "1px solid #333",
              borderRadius: 8,
              padding: "1rem",
              fontSize: 13,
              overflowX: "auto",
              color: "#f87171",
              whiteSpace: "pre-wrap",
              wordBreak: "break-word",
            }}
          >
            {error?.message || "Unknown error"}
            {"\n\n"}
            {error?.stack || "No stack trace"}
            {error?.digest ? `\n\ndigest: ${error.digest}` : ""}
          </pre>
          <button
            onClick={reset}
            style={{
              marginTop: "1rem",
              padding: "0.5rem 1.5rem",
              background: "#7c3aed",
              color: "#fff",
              border: "none",
              borderRadius: 6,
              cursor: "pointer",
              fontSize: 14,
            }}
          >
            Try again
          </button>
        </div>
      </body>
    </html>
  );
}
