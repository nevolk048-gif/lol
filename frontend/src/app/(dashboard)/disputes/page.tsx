"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { AlertTriangle, MessageSquare, Send } from "lucide-react";
import { toast } from "sonner";

interface Dispute {
  id: string;
  transaction_id: string;
  provider_id: string;
  provider_name: string;
  casino_id: string;
  casino_name: string;
  status: string;
  reason: string;
  amount: number;
  currency: string;
  created_at: string;
  updated_at: string;
}

interface DisputeMessage {
  id: string;
  sender_type: string;
  sender_name?: string;
  message: string;
  created_at: string;
}

export default function DisputesPage() {
  const [selectedDispute, setSelectedDispute] = useState<Dispute | null>(null);
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [newMessage, setNewMessage] = useState("");
  const queryClient = useQueryClient();

  // Fetch disputes
  const { data: disputes, isLoading } = useQuery({
    queryKey: ["disputes", statusFilter],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (statusFilter !== "all") params.append("status", statusFilter);

      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/disputes?${params}`);
      if (!res.ok) throw new Error("Failed to fetch disputes");
      return res.json();
    },
  });

  // Fetch dispute messages
  const { data: messages } = useQuery({
    queryKey: ["dispute-messages", selectedDispute?.id],
    queryFn: async () => {
      if (!selectedDispute) return [];
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/disputes/${selectedDispute.id}/messages`);
      if (!res.ok) throw new Error("Failed to fetch messages");
      return res.json();
    },
    enabled: !!selectedDispute,
  });

  // Update status mutation
  const updateStatusMutation = useMutation({
    mutationFn: async ({ disputeId, status }: { disputeId: string; status: string }) => {
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/disputes/${disputeId}/status`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ status }),
      });
      if (!res.ok) throw new Error("Failed to update status");
      return res.json();
    },
    onSuccess: () => {
      toast.success("Статус спора обновлен");
      queryClient.invalidateQueries({ queryKey: ["disputes"] });
      setSelectedDispute(null);
    },
  });

  // Send message mutation
  const sendMessageMutation = useMutation({
    mutationFn: async ({ disputeId, message }: { disputeId: string; message: string }) => {
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/disputes/${disputeId}/messages`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ message }),
      });
      if (!res.ok) throw new Error("Failed to send message");
      return res.json();
    },
    onSuccess: () => {
      toast.success("Сообщение отправлено");
      queryClient.invalidateQueries({ queryKey: ["dispute-messages"] });
      setNewMessage("");
    },
  });

  const getStatusBadge = (status: string) => {
    const variants: Record<string, string> = {
      NEW: "bg-blue-100 text-blue-800",
      UNDER_REVIEW: "bg-yellow-100 text-yellow-800",
      AWAITING_PROVIDER_RESPONSE: "bg-orange-100 text-orange-800",
      MERCHANT_WON: "bg-green-100 text-green-800",
      PROVIDER_WON: "bg-red-100 text-red-800",
      CLOSED: "bg-gray-100 text-gray-800",
    };
    return variants[status] || "bg-gray-100 text-gray-800";
  };

  const getStatusText = (status: string) => {
    const texts: Record<string, string> = {
      NEW: "Новый",
      UNDER_REVIEW: "На рассмотрении",
      AWAITING_PROVIDER_RESPONSE: "Ожидает ответа провайдера",
      MERCHANT_WON: "Выиграл мерчант",
      PROVIDER_WON: "Выиграл провайдер",
      CLOSED: "Закрыт",
    };
    return texts[status] || status;
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Управление спорами</h1>
          <p className="text-muted-foreground">Система разрешения споров между мерчантами и провайдерами</p>
        </div>
      </div>

      {/* Filters */}
      <Card>
        <CardHeader>
          <CardTitle>Фильтры</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex gap-4">
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="w-[200px]">
                <SelectValue placeholder="Статус" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Все</SelectItem>
                <SelectItem value="NEW">Новые</SelectItem>
                <SelectItem value="UNDER_REVIEW">На рассмотрении</SelectItem>
                <SelectItem value="AWAITING_PROVIDER_RESPONSE">Ожидает ответа</SelectItem>
                <SelectItem value="MERCHANT_WON">Мерчант выиграл</SelectItem>
                <SelectItem value="PROVIDER_WON">Провайдер выиграл</SelectItem>
                <SelectItem value="CLOSED">Закрыт</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      {/* Disputes List */}
      <div className="grid gap-4">
        {isLoading && <div>Загрузка...</div>}
        {disputes?.map((dispute: Dispute) => (
          <Card key={dispute.id} className="hover:shadow-md transition-shadow">
            <CardContent className="pt-6">
              <div className="flex items-start justify-between">
                <div className="space-y-2 flex-1">
                  <div className="flex items-center gap-3">
                    <AlertTriangle className="h-5 w-5 text-orange-500" />
                    <h3 className="font-semibold">Спор #{dispute.id.slice(0, 8)}</h3>
                    <Badge className={getStatusBadge(dispute.status)}>
                      {getStatusText(dispute.status)}
                    </Badge>
                  </div>

                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div>
                      <span className="text-muted-foreground">Провайдер:</span>{" "}
                      <span className="font-medium">{dispute.provider_name}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Мерчант:</span>{" "}
                      <span className="font-medium">{dispute.casino_name}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Сумма:</span>{" "}
                      <span className="font-medium">{dispute.amount} {dispute.currency}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Создан:</span>{" "}
                      <span className="font-medium">{new Date(dispute.created_at).toLocaleString('ru')}</span>
                    </div>
                  </div>

                  <div className="pt-2">
                    <span className="text-muted-foreground text-sm">Причина:</span>
                    <p className="text-sm mt-1">{dispute.reason}</p>
                  </div>
                </div>

                <div className="flex gap-2 ml-4">
                  <Dialog>
                    <DialogTrigger asChild>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setSelectedDispute(dispute)}
                      >
                        <MessageSquare className="h-4 w-4 mr-1" />
                        Сообщения
                      </Button>
                    </DialogTrigger>
                    <DialogContent className="max-w-2xl max-h-[600px]">
                      <DialogHeader>
                        <DialogTitle>Спор #{dispute.id.slice(0, 8)} - Сообщения</DialogTitle>
                      </DialogHeader>

                      <div className="space-y-4">
                        {/* Messages */}
                        <div className="h-[300px] overflow-y-auto space-y-3 border rounded-lg p-4">
                          {messages?.map((msg: DisputeMessage) => (
                            <div key={msg.id} className={`p-3 rounded ${msg.sender_type === 'ADMIN' ? 'bg-blue-50' : 'bg-gray-50'}`}>
                              <div className="flex justify-between text-xs text-muted-foreground mb-1">
                                <span className="font-medium">{msg.sender_type}</span>
                                <span>{new Date(msg.created_at).toLocaleString('ru')}</span>
                              </div>
                              <p className="text-sm">{msg.message}</p>
                            </div>
                          ))}
                          {!messages?.length && (
                            <div className="text-center text-muted-foreground">Нет сообщений</div>
                          )}
                        </div>

                        {/* Send message */}
                        <div className="flex gap-2">
                          <Textarea
                            placeholder="Введите сообщение..."
                            value={newMessage}
                            onChange={(e) => setNewMessage(e.target.value)}
                            rows={3}
                          />
                          <Button
                            onClick={() => {
                              if (selectedDispute && newMessage.trim()) {
                                sendMessageMutation.mutate({
                                  disputeId: selectedDispute.id,
                                  message: newMessage,
                                });
                              }
                            }}
                            disabled={!newMessage.trim() || sendMessageMutation.isPending}
                          >
                            <Send className="h-4 w-4" />
                          </Button>
                        </div>

                        {/* Update status */}
                        <div className="border-t pt-4">
                          <label className="text-sm font-medium mb-2 block">Изменить статус</label>
                          <div className="flex gap-2">
                            <Select
                              onValueChange={(status) => {
                                if (selectedDispute) {
                                  updateStatusMutation.mutate({
                                    disputeId: selectedDispute.id,
                                    status,
                                  });
                                }
                              }}
                            >
                              <SelectTrigger>
                                <SelectValue placeholder="Выберите статус" />
                              </SelectTrigger>
                              <SelectContent>
                                <SelectItem value="UNDER_REVIEW">На рассмотрении</SelectItem>
                                <SelectItem value="AWAITING_PROVIDER_RESPONSE">Ожидает ответа провайдера</SelectItem>
                                <SelectItem value="MERCHANT_WON">Мерчант выиграл</SelectItem>
                                <SelectItem value="PROVIDER_WON">Провайдер выиграл</SelectItem>
                                <SelectItem value="CLOSED">Закрыт</SelectItem>
                              </SelectContent>
                            </Select>
                          </div>
                        </div>
                      </div>
                    </DialogContent>
                  </Dialog>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
