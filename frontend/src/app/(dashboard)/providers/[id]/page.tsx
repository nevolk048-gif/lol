"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useParams } from "next/navigation";
import { api } from "@/services/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { StatCard } from "@/components/widgets/stat-card";
import { StatCardSkeleton } from "@/components/ui/skeleton";
import { TrafficControl } from "@/components/traffic-control";
import { DollarSign, ArrowLeftRight, Clock, Activity, PlayCircle, PauseCircle, Ban } from "lucide-react";
import { toast } from "sonner";

export default function ProviderDetailPage() {
  const params = useParams();
  const id = params.id as string;
  const queryClient = useQueryClient();

  const { data: provider, isLoading } = useQuery({
    queryKey: ["provider", id],
    queryFn: () => api.getProvider(id),
  });

  const { data: requisites } = useQuery({
    queryKey: ["requisites", id],
    queryFn: () => api.getRequisites(id),
    select: (data) => Array.isArray(data) ? data : [],
  });

  const updateStatusMutation = useMutation({
    mutationFn: (status: string) => api.updateProviderStatus(id, status),
    onSuccess: () => {
      toast.success("Provider status updated");
      queryClient.invalidateQueries({ queryKey: ["provider", id] });
      queryClient.invalidateQueries({ queryKey: ["providers"] });
    },
    onError: () => toast.error("Failed to update status"),
  });

  if (isLoading) {
    return (
      <div className="grid gap-4 md:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => <StatCardSkeleton key={i} />)}
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold">{provider?.name}</h1>
              <Badge status={provider?.status}>{provider?.status}</Badge>
            </div>
            <p className="text-muted-foreground font-mono text-sm mt-1">{provider?.api_key}</p>
          </div>
        </div>
        <div className="flex gap-2">
          {provider?.status !== "ACTIVE" && (
            <Button
              size="sm"
              variant="outline"
              onClick={() => updateStatusMutation.mutate("ACTIVE")}
              disabled={updateStatusMutation.isPending}
            >
              <PlayCircle className="h-4 w-4 mr-2" />
              Activate
            </Button>
          )}
          {provider?.status === "ACTIVE" && (
            <Button
              size="sm"
              variant="outline"
              onClick={() => updateStatusMutation.mutate("INACTIVE")}
              disabled={updateStatusMutation.isPending}
            >
              <PauseCircle className="h-4 w-4 mr-2" />
              Pause Traffic
            </Button>
          )}
          <Button
            size="sm"
            variant="destructive"
            onClick={() => {
              if (confirm("Block this provider? It will stop receiving all traffic.")) {
                updateStatusMutation.mutate("BLOCKED");
              }
            }}
            disabled={updateStatusMutation.isPending || provider?.status === "BLOCKED"}
          >
            <Ban className="h-4 w-4 mr-2" />
            Block
          </Button>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-4">
        <StatCard title="Turnover" value={provider?.turnover ?? 0} format="currency" icon={DollarSign} />
        <StatCard title="Transactions" value={provider?.transaction_count ?? 0} format="number" icon={ArrowLeftRight} />
        <StatCard title="Avg Response" value={`${provider?.avg_response_ms?.toFixed(0) ?? 0}ms`} icon={Clock} />
        <StatCard title="Conversion" value={provider?.conversion_rate ?? 0} format="percent" icon={Activity} />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader><CardTitle>API Settings</CardTitle></CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex justify-between"><span className="text-muted-foreground">API Key</span><code>{provider?.api_key}</code></div>
            <div className="flex justify-between"><span className="text-muted-foreground">Webhook</span><span>{provider?.webhook_url || "Not configured"}</span></div>
            <div className="flex justify-between"><span className="text-muted-foreground">Environment</span><span>{provider?.is_sandbox ? "Sandbox" : "Production"}</span></div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>Requisites ({requisites?.length ?? 0})</CardTitle></CardHeader>
          <CardContent>
            <div className="space-y-2">
              {requisites?.map((r) => (
                <div key={r.id} className="flex items-center justify-between rounded-lg border border-border p-3 text-sm">
                  <div>
                    <p className="font-medium">{r.bank_name}</p>
                    <p className="text-muted-foreground">{r.account_number}</p>
                  </div>
                  <Badge status={r.status}>{r.status}</Badge>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Traffic Control */}
      <TrafficControl providerId={id} currentStatus={provider?.traffic_enabled} />
    </div>
  );
}
