"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/services/api";
import { StatCard } from "@/components/widgets/stat-card";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ProviderLoadChart } from "@/components/charts/charts";
import { useWebSocket } from "@/hooks/use-websocket";
import { Activity, Wifi, AlertTriangle, Clock, Server } from "lucide-react";

export default function MonitoringPage() {
  const queryClient = useQueryClient();

  useWebSocket(() => {
    queryClient.invalidateQueries({ queryKey: ["monitoring"] });
  });

  const { data } = useQuery({
    queryKey: ["monitoring"],
    queryFn: () => api.getMonitoring(),
    refetchInterval: 5000,
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2">
        <div className="h-2 w-2 rounded-full bg-success animate-pulse" />
        <h1 className="text-2xl font-bold">Monitoring</h1>
        <span className="text-sm text-muted-foreground">Live</span>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
        <StatCard title="RPS" value={data?.rps?.toFixed(2) ?? "0"} icon={Activity} />
        <StatCard title="WS Connections" value={data?.ws_connections ?? 0} format="number" icon={Wifi} />
        <StatCard title="Error Rate" value={data?.error_rate ?? 0} format="percent" icon={AlertTriangle} />
        <StatCard title="Avg Latency" value={`${data?.avg_latency_ms?.toFixed(0) ?? 0}ms`} icon={Clock} />
        <StatCard title="Active Connections" value={data?.active_connections ?? 0} format="number" icon={Server} />
      </div>

      <Card>
        <CardHeader><CardTitle>Provider Load (Last Hour)</CardTitle></CardHeader>
        <CardContent>
          <ProviderLoadChart data={data?.provider_load ?? []} />
        </CardContent>
      </Card>
    </div>
  );
}
