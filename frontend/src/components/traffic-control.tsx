"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { AlertTriangle, CheckCircle, XCircle, History } from "lucide-react";
import { toast } from "sonner";

interface TrafficControlProps {
  providerId: string;
  currentStatus?: boolean;
}

export function TrafficControl({ providerId, currentStatus = true }: TrafficControlProps) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [reason, setReason] = useState("");
  const queryClient = useQueryClient();

  // Fetch traffic status
  const { data: trafficStatus, isLoading } = useQuery({
    queryKey: ["traffic-status", providerId],
    queryFn: async () => {
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/traffic/providers/${providerId}/status`);
      if (!res.ok) throw new Error("Failed to fetch traffic status");
      return res.json();
    },
  });

  // Fetch traffic history
  const { data: trafficHistory } = useQuery({
    queryKey: ["traffic-history", providerId],
    queryFn: async () => {
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/traffic/providers/${providerId}/history`);
      if (!res.ok) throw new Error("Failed to fetch traffic history");
      return res.json();
    },
  });

  // Update traffic mutation
  const updateTrafficMutation = useMutation({
    mutationFn: async ({ enabled, reason }: { enabled: boolean; reason?: string }) => {
      const url = enabled
        ? `${process.env.NEXT_PUBLIC_API_URL}/traffic/providers/${providerId}/enable`
        : `${process.env.NEXT_PUBLIC_API_URL}/traffic/providers/${providerId}/disable`;

      const res = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ reason }),
      });

      if (!res.ok) throw new Error("Failed to update traffic");
      return res.json();
    },
    onSuccess: (_, variables) => {
      toast.success(variables.enabled ? "Трафик включен" : "Трафик отключен");
      queryClient.invalidateQueries({ queryKey: ["traffic-status", providerId] });
      queryClient.invalidateQueries({ queryKey: ["traffic-history", providerId] });
      queryClient.invalidateQueries({ queryKey: ["provider", providerId] });
      setIsDialogOpen(false);
      setReason("");
    },
    onError: () => {
      toast.error("Не удалось обновить трафик");
    },
  });

  const handleToggle = (checked: boolean) => {
    if (!checked) {
      // Если отключаем, открываем диалог для причины
      setIsDialogOpen(true);
    } else {
      // Если включаем, делаем это сразу
      updateTrafficMutation.mutate({ enabled: true });
    }
  };

  const handleConfirmDisable = () => {
    if (!reason.trim()) {
      toast.error("Укажите причину отключения");
      return;
    }
    updateTrafficMutation.mutate({ enabled: false, reason });
  };

  const enabled = trafficStatus?.enabled ?? currentStatus;

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span className="flex items-center gap-2">
              {enabled ? (
                <CheckCircle className="h-5 w-5 text-green-500" />
              ) : (
                <XCircle className="h-5 w-5 text-red-500" />
              )}
              Управление трафиком
            </span>
            <Badge className={enabled ? "bg-green-500/10 text-green-500 border-green-500/20" : "bg-red-500/10 text-red-500 border-red-500/20"}>
              {enabled ? "🟢 Включен" : "🔴 Отключен"}
            </Badge>
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <Label htmlFor="traffic-toggle">Прием трафика</Label>
              <p className="text-sm text-muted-foreground">
                {enabled
                  ? "Провайдер получает новые транзакции"
                  : "Провайдер не участвует в маршрутизации"}
              </p>
            </div>
            <Switch
              id="traffic-toggle"
              checked={enabled}
              onCheckedChange={handleToggle}
              disabled={isLoading || updateTrafficMutation.isPending}
            />
          </div>

          {!enabled && trafficStatus?.disabled_reason && (
            <div className="bg-red-50 border border-red-200 rounded-lg p-3">
              <div className="flex items-start gap-2">
                <AlertTriangle className="h-4 w-4 text-red-500 mt-0.5" />
                <div className="flex-1">
                  <p className="text-sm font-medium text-red-900">Причина блокировки</p>
                  <p className="text-sm text-red-700 mt-1">{trafficStatus.disabled_reason}</p>
                  {trafficStatus.disabled_at && (
                    <p className="text-xs text-red-600 mt-1">
                      Отключен: {new Date(trafficStatus.disabled_at).toLocaleString('ru')}
                    </p>
                  )}
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Traffic History */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <History className="h-5 w-5" />
            История изменений
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {trafficHistory?.map((item: { id: string; action: string; reason?: string; created_at: string }) => (
              <div key={item.id} className="flex items-start gap-3 pb-3 border-b last:border-0">
                <div className={`p-2 rounded-full ${item.action === 'ENABLED' ? 'bg-green-100' : 'bg-red-100'}`}>
                  {item.action === 'ENABLED' ? (
                    <CheckCircle className="h-4 w-4 text-green-600" />
                  ) : (
                    <XCircle className="h-4 w-4 text-red-600" />
                  )}
                </div>
                <div className="flex-1">
                  <div className="flex items-center justify-between">
                    <span className="font-medium text-sm">
                      {item.action === 'ENABLED' ? 'Трафик включен' : 'Трафик отключен'}
                    </span>
                    <span className="text-xs text-muted-foreground">
                      {new Date(item.created_at).toLocaleString('ru')}
                    </span>
                  </div>
                  {item.reason && (
                    <p className="text-sm text-muted-foreground mt-1">{item.reason}</p>
                  )}
                </div>
              </div>
            ))}
            {!trafficHistory?.length && (
              <p className="text-sm text-muted-foreground text-center py-4">
                История изменений пуста
              </p>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Disable Dialog */}
      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Отключить трафик провайдера</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="reason">Причина отключения *</Label>
              <Textarea
                id="reason"
                placeholder="Укажите причину отключения трафика..."
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                rows={4}
              />
              <p className="text-xs text-muted-foreground">
                Причина будет видна в истории изменений и логах
              </p>
            </div>
            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-3">
              <p className="text-sm text-yellow-800">
                <strong>Внимание:</strong> После отключения трафика новые транзакции не будут направляться этому провайдеру. Существующие транзакции продолжат обрабатываться.
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsDialogOpen(false)}>
              Отмена
            </Button>
            <Button
              variant="destructive"
              onClick={handleConfirmDisable}
              disabled={!reason.trim() || updateTrafficMutation.isPending}
            >
              {updateTrafficMutation.isPending ? "Отключение..." : "Отключить трафик"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
