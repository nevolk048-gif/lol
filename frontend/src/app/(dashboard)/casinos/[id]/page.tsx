"use client";

import { useQuery } from "@tanstack/react-query";
import { useParams } from "next/navigation";
import { api } from "@/services/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { StatCard } from "@/components/widgets/stat-card";
import { DollarSign, ArrowLeftRight } from "lucide-react";

export default function CasinoDetailPage() {
  const params = useParams();
  const id = params.id as string;

  const { data: casino } = useQuery({
    queryKey: ["casino", id],
    queryFn: () => api.getCasino(id),
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <h1 className="text-2xl font-bold">{casino?.name}</h1>
        <Badge status={casino?.status}>{casino?.status}</Badge>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <StatCard title="Turnover" value={casino?.turnover ?? 0} format="currency" icon={DollarSign} />
        <StatCard title="Transactions" value={casino?.transaction_count ?? 0} format="number" icon={ArrowLeftRight} />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader><CardTitle>API Settings</CardTitle></CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex justify-between"><span className="text-muted-foreground">API Key</span><code className="text-xs">{casino?.api_key}</code></div>
            <div className="flex justify-between"><span className="text-muted-foreground">Webhook URL</span><span>{casino?.webhook_url || "Not configured"}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">Environment</span><span>{casino?.is_sandbox ? "Sandbox" : "Production"}</span></div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>Webhook Configuration</CardTitle></CardHeader>
          <CardContent className="text-sm text-muted-foreground">
            <p>POST notifications sent on transaction status changes.</p>
            <p className="mt-2">HMAC SHA256 signature in X-Signature header.</p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
