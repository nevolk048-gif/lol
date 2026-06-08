"use client";

import { useEffect, useRef, useCallback } from "react";
import { useAuthStore } from "@/stores/auth-store";
import type { WSEvent } from "@/types";

// Автоматически определяем WebSocket URL на основе API URL
const getWsUrl = () => {
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
  // Преобразуем http(s):// в ws(s)://
  return apiUrl.replace(/^http/, 'ws');
};

const WS_URL = getWsUrl();

export function useWebSocket(onEvent?: (event: WSEvent) => void) {
  const wsRef = useRef<WebSocket | null>(null);
  const { accessToken } = useAuthStore();

  const connect = useCallback(() => {
    if (!accessToken || wsRef.current?.readyState === WebSocket.OPEN) return;

    const ws = new WebSocket(`${WS_URL}/ws?token=${accessToken}`);
    wsRef.current = ws;

    ws.onmessage = (msg) => {
      try {
        const event = JSON.parse(msg.data) as WSEvent;
        onEvent?.(event);
      } catch {
        /* ignore */
      }
    };

    ws.onclose = () => {
      setTimeout(connect, 3000);
    };

    ws.onerror = () => {
      // Закрываем соединение при ошибке, чтобы onclose мог переподключиться
      ws.close();
    };
  }, [accessToken, onEvent]);

  useEffect(() => {
    connect();
    return () => {
      wsRef.current?.close();
    };
  }, [connect]);
}
