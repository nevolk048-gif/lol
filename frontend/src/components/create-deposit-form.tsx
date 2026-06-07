"use client";

import { useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { api } from "@/services/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { toast } from "sonner";
import { Wallet } from "lucide-react";

export function CreateDepositForm() {
  const [formData, setFormData] = useState({
    casino_id: "",
    amount: "100000", // в копейках, мин 1000₽
    merchant_customer_id: "",
    payment_method: "auto",
    description: "",
    return_url: "https://peaceful-hope-production-bb2d.up.railway.app/pay/success",
    metadata_key: "",
    metadata_value: "",
  });

  const { data: casinos } = useQuery({
    queryKey: ["casinos"],
    queryFn: () => api.getCasinos(),
    select: (data) => Array.isArray(data) ? data : [],
  });

  const createMutation = useMutation({
    mutationFn: async (data: typeof formData) => {
      // Получаем API ключ казино для авторизации
      const casino = casinos?.find(c => c.id === data.casino_id);
      if (!casino) {
        throw new Error("Casino not found");
      }

      const metadata: Record<string, string> = {};
      if (data.metadata_key && data.metadata_value) {
        metadata[data.metadata_key] = data.metadata_value;
      }

      const payload = {
        amount: parseInt(data.amount),
        merchant_customer_id: data.merchant_customer_id,
        payment_method: data.payment_method === "auto" ? undefined : data.payment_method,
        description: data.description || undefined,
        return_url: data.return_url,
        metadata: Object.keys(metadata).length > 0 ? metadata : undefined,
      };

      // Вызываем API создания депозита от имени казино
      const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"}/api/v1/deposit`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-API-Key": casino.api_key,
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error?.message || "Failed to create deposit");
      }

      return response.json();
    },
    onSuccess: (result) => {
      toast.success(`Deposit created! Transaction ID: ${result.data?.transaction_id || "N/A"}`);
      console.log("Deposit result:", result);
      // Опционально: открыть HPP в новой вкладке
      if (result.data?.hosted_payment_page_url) {
        window.open(result.data.hosted_payment_page_url, "_blank");
      }
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to create deposit");
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.casino_id) {
      toast.error("Please select a casino");
      return;
    }
    if (!formData.merchant_customer_id) {
      toast.error("Please provide merchant customer ID");
      return;
    }
    createMutation.mutate(formData);
  };

  const amountInRubles = (parseInt(formData.amount) || 0) / 100;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Wallet className="h-5 w-5" />
          Create Deposit (Casino API)
        </CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <Label>Casino *</Label>
            <Select value={formData.casino_id} onValueChange={(v) => setFormData({ ...formData, casino_id: v })}>
              <SelectTrigger>
                <SelectValue placeholder="Select casino" />
              </SelectTrigger>
              <SelectContent>
                {casinos?.map((c) => (
                  <SelectItem key={c.id} value={c.id}>
                    {c.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label>Amount (kopecks) * <span className="text-muted-foreground text-xs">≈ {amountInRubles.toFixed(2)} ₽</span></Label>
              <Input
                type="number"
                min="100000"
                max="50000000"
                value={formData.amount}
                onChange={(e) => setFormData({ ...formData, amount: e.target.value })}
                placeholder="100000 (min 1000₽)"
                required
              />
            </div>
            <div>
              <Label>Payment Method *</Label>
              <Select value={formData.payment_method} onValueChange={(v) => setFormData({ ...formData, payment_method: v })}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="auto">auto (Recommended)</SelectItem>
                  <SelectItem value="card">card</SelectItem>
                  <SelectItem value="sbp">sbp</SelectItem>
                  <SelectItem value="mobcom">mobcom</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <div>
            <Label>Merchant Customer ID * <span className="text-muted-foreground text-xs">(for Payer Affinity)</span></Label>
            <Input
              value={formData.merchant_customer_id}
              onChange={(e) => setFormData({ ...formData, merchant_customer_id: e.target.value })}
              placeholder="customer_12345"
              maxLength={128}
              required
            />
          </div>

          <div>
            <Label>Description</Label>
            <Input
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              placeholder="Deposit to balance"
            />
          </div>

          <div>
            <Label>Return URL</Label>
            <Input
              value={formData.return_url}
              onChange={(e) => setFormData({ ...formData, return_url: e.target.value })}
              placeholder="https://your-site.com/pay/success"
            />
          </div>

          <div>
            <Label>Metadata (optional)</Label>
            <div className="grid grid-cols-2 gap-2">
              <Input
                value={formData.metadata_key}
                onChange={(e) => setFormData({ ...formData, metadata_key: e.target.value })}
                placeholder="Key (e.g. order_id)"
                maxLength={64}
              />
              <Input
                value={formData.metadata_value}
                onChange={(e) => setFormData({ ...formData, metadata_value: e.target.value })}
                placeholder="Value (e.g. 100293)"
                maxLength={240}
              />
            </div>
          </div>

          <Button type="submit" disabled={createMutation.isPending} className="w-full">
            {createMutation.isPending ? "Creating..." : "Create Deposit"}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
