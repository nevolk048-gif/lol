"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { type ColumnDef } from "@tanstack/react-table";
import { useMemo, useState } from "react";
import { api } from "@/services/api";
import { DataTable } from "@/components/widgets/data-table";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { TableSkeleton } from "@/components/ui/skeleton";
import { formatCurrency, formatDate, truncate } from "@/lib/utils";
import type { Dispute, Transaction, TransactionStatus } from "@/types";
import { Filter, PlayCircle, AlertTriangle } from "lucide-react";
import { toast } from "sonner";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";

const statuses: TransactionStatus[] = [
  "NEW", "ASSIGNED", "WAITING_PAYMENT", "PAID", "EXPIRED", "CANCELLED",
];

export default function TransactionsPage() {
  const [statusFilter, setStatusFilter] = useState("");
  const [countryFilter, setCountryFilter] = useState("");
  const [isDisputeDialogOpen, setIsDisputeDialogOpen] = useState(false);
  const [selectedTransaction, setSelectedTransaction] = useState<string | null>(null);
  const [disputeReason, setDisputeReason] = useState("");
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ["transactions", statusFilter, countryFilter],
    queryFn: () =>
      api.getTransactions({
        ...(statusFilter && { status: statusFilter }),
        ...(countryFilter && { country: countryFilter }),
        per_page: "50",
      }),
  });

  // Споры по транзакциям — чтобы показывать индикатор спора в карточке транзакции
  const { data: disputesData } = useQuery({
    queryKey: ["disputes"],
    queryFn: () => api.getDisputes() as Promise<Dispute[]>,
  });

  const disputeByTx = useMemo(() => {
    const map = new Map<string, Dispute>();
    const list = Array.isArray(disputesData) ? disputesData : [];
    list.forEach((d) => {
      // оставляем самый свежий спор по транзакции (список уже отсортирован по дате убыв.)
      if (d?.transaction_id && !map.has(d.transaction_id)) map.set(d.transaction_id, d);
    });
    return map;
  }, [disputesData]);

  const disputeStatusText: Record<string, string> = {
    NEW: "Новый",
    UNDER_REVIEW: "На рассмотрении",
    AWAITING_PROVIDER_RESPONSE: "Ожидает ответа",
    MERCHANT_WON: "Мерчант выиграл",
    PROVIDER_WON: "Провайдер выиграл",
    CLOSED: "Закрыт",
  };

  const simulatePaymentMutation = useMutation({
    mutationFn: (transactionId: string) => api.sandboxSimulatePayment(transactionId),
    onSuccess: () => {
      toast.success("✅ Оплата успешно симулирована");
      queryClient.invalidateQueries({ queryKey: ["transactions"] });
    },
    onError: (error: Error) => {
      console.error("Simulate payment error:", error);
      const errorMessage = error?.message || "Ошибка симуляции оплаты";
      toast.error(`❌ Ошибка: ${errorMessage}`);
    },
  });

  const createDisputeMutation = useMutation({
    mutationFn: (data: { transaction_id: string; reason: string }) => api.createDispute(data),
    onSuccess: () => {
      toast.success("✅ Спор успешно создан. Трафик провайдера автоматически заблокирован.");
      queryClient.invalidateQueries({ queryKey: ["transactions"] });
      queryClient.invalidateQueries({ queryKey: ["disputes"] });
      setIsDisputeDialogOpen(false);
      setDisputeReason("");
      setSelectedTransaction(null);
    },
    onError: () => {
      toast.error("❌ Ошибка создания спора");
    },
  });

  const handleCreateDispute = () => {
    if (!selectedTransaction || !disputeReason.trim()) {
      toast.error("Укажите причину спора");
      return;
    }
    createDisputeMutation.mutate({ transaction_id: selectedTransaction, reason: disputeReason });
  };

  const columns: ColumnDef<Transaction>[] = [
    {
      accessorKey: "id",
      header: "ID",
      cell: ({ row }) => (
        <span className="font-mono text-xs">{truncate(row.original.id, 12)}</span>
      ),
    },
    { accessorKey: "casino_name", header: "Casino" },
    { accessorKey: "provider_name", header: "Provider" },
    { accessorKey: "requisite_bank", header: "Requisite" },
    { accessorKey: "country", header: "Country" },
    {
      accessorKey: "amount",
      header: "Amount",
      cell: ({ row }) => formatCurrency(row.original.amount, row.original.currency),
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <Badge status={row.original.status}>{row.original.status.replace("_", " ")}</Badge>,
    },
    {
      id: "dispute",
      header: "Спор",
      cell: ({ row }) => {
        const dispute = disputeByTx.get(row.original.id);
        if (!dispute) return <span className="text-muted-foreground text-xs">—</span>;
        return (
          <span
            className="inline-flex items-center gap-1 rounded-full bg-orange-100 px-2 py-0.5 text-xs font-medium text-orange-800"
            title={`Спор #${String(dispute.id ?? "").slice(0, 8)}: ${dispute.reason ?? ""}`}
          >
            <AlertTriangle className="h-3 w-3" />
            {disputeStatusText[dispute.status] ?? dispute.status}
          </span>
        );
      },
    },
    {
      accessorKey: "created_at",
      header: "Date",
      cell: ({ row }) => formatDate(row.original.created_at),
    },
    {
      id: "actions",
      header: "Actions",
      cell: ({ row }) => {
        const canSimulate = row.original.is_sandbox &&
          (row.original.status === "NEW" || row.original.status === "ASSIGNED" || row.original.status === "WAITING_PAYMENT");

        const canDispute = row.original.is_sandbox && row.original.status === "PAID";

        if (!canSimulate && !canDispute) return null;

        return (
          <div className="flex gap-1">
            {canSimulate && (
              <Button
                size="sm"
                variant="ghost"
                onClick={() => simulatePaymentMutation.mutate(row.original.id)}
                disabled={simulatePaymentMutation.isPending}
                title="Симулировать оплату"
              >
                <PlayCircle className="h-4 w-4 text-green-600" />
              </Button>
            )}
            {canDispute && (
              <Button
                size="sm"
                variant="ghost"
                onClick={() => {
                  setSelectedTransaction(row.original.id);
                  setIsDisputeDialogOpen(true);
                }}
                title="Создать спор"
              >
                <AlertTriangle className="h-4 w-4 text-orange-600" />
              </Button>
            )}
          </div>
        );
      },
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Transactions</h1>
          <p className="text-muted-foreground">All deposit requests and their routing status</p>
        </div>
      </div>

      <div className="flex flex-wrap gap-3">
        <div className="flex items-center gap-2">
          <Filter className="h-4 w-4 text-muted-foreground" />
          <select
            className="h-10 rounded-lg border border-border bg-background px-3 text-sm"
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
          >
            <option value="">All Statuses</option>
            {statuses.map((s) => (
              <option key={s} value={s}>{s.replace("_", " ")}</option>
            ))}
          </select>
        </div>
        <Input
          placeholder="Filter by country (US, DE...)"
          className="w-48"
          value={countryFilter}
          onChange={(e) => setCountryFilter(e.target.value.toUpperCase())}
        />
        <Button variant="outline" onClick={() => { setStatusFilter(""); setCountryFilter(""); }}>
          Clear filters
        </Button>
      </div>

      {isLoading ? <TableSkeleton /> : <DataTable data={data ?? []} columns={columns} />}

      {/* Dispute Dialog */}
      <Dialog open={isDisputeDialogOpen} onOpenChange={setIsDisputeDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Создать спор (Dispute)</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="dispute-reason">Причина спора *</Label>
              <Textarea
                id="dispute-reason"
                placeholder="Опишите причину спора (например: мошенническая транзакция, charгeback от клиента, несоответствие суммы)..."
                value={disputeReason}
                onChange={(e) => setDisputeReason(e.target.value)}
                rows={5}
              />
              <p className="text-xs text-muted-foreground">
                Причина будет видна провайдеру и мерчанту
              </p>
            </div>
            <div className="bg-orange-50 border border-orange-200 rounded-lg p-3">
              <p className="text-sm text-orange-800">
                <strong>⚠️ Внимание:</strong> При создании спора трафик провайдера будет автоматически заблокирован до разрешения ситуации.
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => {
              setIsDisputeDialogOpen(false);
              setDisputeReason("");
              setSelectedTransaction(null);
            }}>
              Отмена
            </Button>
            <Button
              variant="destructive"
              onClick={handleCreateDispute}
              disabled={!disputeReason.trim() || createDisputeMutation.isPending}
            >
              {createDisputeMutation.isPending ? "Создание..." : "Создать спор"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
