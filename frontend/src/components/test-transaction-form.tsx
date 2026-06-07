"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/services/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { toast } from "sonner";
import { Send } from "lucide-react";

export function TestTransactionForm() {
  const queryClient = useQueryClient();
  const [formData, setFormData] = useState({
    casino_id: "",
    provider_id: "",
    amount: "100",
    currency: "USD",
    country: "US",
    player_id: "",
  });

  const { data: casinos } = useQuery({
    queryKey: ["casinos"],
    queryFn: () => api.getCasinos(),
    select: (data) => Array.isArray(data) ? data : [],
  });

  const { data: providers } = useQuery({
    queryKey: ["providers"],
    queryFn: () => api.getProviders(),
    select: (data) => Array.isArray(data) ? data : [],
  });

  const createMutation = useMutation({
    mutationFn: (data: typeof formData) => {
      const payload = {
        casino_id: data.casino_id,
        amount: parseFloat(data.amount),
        currency: data.currency,
        country: data.country,
        ...(data.provider_id && { provider_id: data.provider_id }),
        ...(data.player_id && { player_id: data.player_id }),
      };
      return api.createTestTransaction(payload);
    },
    onSuccess: (transaction) => {
      toast.success(`Transaction created: ${transaction.id}`);
      queryClient.invalidateQueries({ queryKey: ["transactions"] });
      setFormData({ casino_id: "", provider_id: "", amount: "100", currency: "USD", country: "US", player_id: "" });
    },
    onError: (error: any) => {
      toast.error(error.message || "Failed to create transaction");
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.casino_id) {
      toast.error("Please select a casino");
      return;
    }
    createMutation.mutate(formData);
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Send className="h-5 w-5" />
          Create Test Transaction
        </CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
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
            <div>
              <Label>Provider (optional)</Label>
              <Select value={formData.provider_id} onValueChange={(v) => setFormData({ ...formData, provider_id: v })}>
                <SelectTrigger>
                  <SelectValue placeholder="Auto-assign" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">Auto-assign</SelectItem>
                  {providers?.map((p) => (
                    <SelectItem key={p.id} value={p.id}>
                      {p.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="grid grid-cols-3 gap-4">
            <div>
              <Label>Amount *</Label>
              <Input
                type="number"
                step="0.01"
                value={formData.amount}
                onChange={(e) => setFormData({ ...formData, amount: e.target.value })}
                required
              />
            </div>
            <div>
              <Label>Currency *</Label>
              <Select value={formData.currency} onValueChange={(v) => setFormData({ ...formData, currency: v })}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="USD">USD</SelectItem>
                  <SelectItem value="EUR">EUR</SelectItem>
                  <SelectItem value="RUB">RUB</SelectItem>
                  <SelectItem value="KZT">KZT</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>Country *</Label>
              <Select value={formData.country} onValueChange={(v) => setFormData({ ...formData, country: v })}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="US">US</SelectItem>
                  <SelectItem value="RU">RU</SelectItem>
                  <SelectItem value="KZ">KZ</SelectItem>
                  <SelectItem value="UA">UA</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <div>
            <Label>Player ID (optional)</Label>
            <Input
              value={formData.player_id}
              onChange={(e) => setFormData({ ...formData, player_id: e.target.value })}
              placeholder="player_12345"
            />
          </div>

          <Button type="submit" disabled={createMutation.isPending} className="w-full">
            {createMutation.isPending ? "Creating..." : "Create Test Transaction"}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
