"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/services/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { LimitProgress } from "@/components/ui/progress";
import { TableSkeleton, EmptyState } from "@/components/ui/skeleton";
import { formatCurrency } from "@/lib/utils";
import { CreditCard, Plus } from "lucide-react";
import { toast } from "sonner";

export default function RequisitesPage() {
  const [showCreate, setShowCreate] = useState(false);
  const [formData, setFormData] = useState({
    provider_id: "",
    bank_name: "",
    holder_name: "",
    account_number: "",
    currency: "USD",
    country: "US",
    daily_limit: 100000,
    is_sandbox: false,
  });

  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ["requisites"],
    queryFn: () => api.getRequisites(),
  });

  const { data: providers } = useQuery({
    queryKey: ["providers"],
    queryFn: () => api.getProviders(),
  });

  const createMutation = useMutation({
    mutationFn: (data: typeof formData) => api.createRequisite(data),
    onSuccess: () => {
      toast.success("Requisite created successfully");
      queryClient.invalidateQueries({ queryKey: ["requisites"] });
      setShowCreate(false);
      setFormData({
        provider_id: "",
        bank_name: "",
        holder_name: "",
        account_number: "",
        currency: "USD",
        country: "US",
        daily_limit: 100000,
        is_sandbox: false,
      });
    },
    onError: () => toast.error("Failed to create requisite"),
  });

  if (isLoading) return <TableSkeleton rows={6} />;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Requisites</h1>
          <p className="text-muted-foreground">Bank accounts and daily limit management</p>
        </div>
        <Button onClick={() => setShowCreate(true)}>
          <Plus className="h-4 w-4 mr-2" />
          Create Requisite
        </Button>
      </div>

      {showCreate && (
        <Card>
          <CardHeader>
            <CardTitle>Create New Requisite</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={(e) => { e.preventDefault(); createMutation.mutate(formData); }} className="space-y-4">
              <div>
                <label className="text-sm font-medium">Provider</label>
                <select
                  className="w-full h-10 rounded-lg border border-border bg-background px-3 text-sm"
                  value={formData.provider_id}
                  onChange={(e) => setFormData({ ...formData, provider_id: e.target.value })}
                  required
                >
                  <option value="">Select provider</option>
                  {Array.isArray(providers) && providers.map((p) => (
                    <option key={p.id} value={p.id}>{p.name}</option>
                  ))}
                </select>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-sm font-medium">Bank Name</label>
                  <Input
                    value={formData.bank_name}
                    onChange={(e) => setFormData({ ...formData, bank_name: e.target.value })}
                    placeholder="Bank name"
                    required
                  />
                </div>
                <div>
                  <label className="text-sm font-medium">Holder Name</label>
                  <Input
                    value={formData.holder_name}
                    onChange={(e) => setFormData({ ...formData, holder_name: e.target.value })}
                    placeholder="Account holder"
                    required
                  />
                </div>
              </div>
              <div>
                <label className="text-sm font-medium">Account Number</label>
                <Input
                  value={formData.account_number}
                  onChange={(e) => setFormData({ ...formData, account_number: e.target.value })}
                  placeholder="****1234"
                  required
                />
              </div>
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="text-sm font-medium">Currency</label>
                  <Input
                    value={formData.currency}
                    onChange={(e) => setFormData({ ...formData, currency: e.target.value })}
                    placeholder="USD"
                    maxLength={3}
                    required
                  />
                </div>
                <div>
                  <label className="text-sm font-medium">Country</label>
                  <Input
                    value={formData.country}
                    onChange={(e) => setFormData({ ...formData, country: e.target.value })}
                    placeholder="US"
                    maxLength={2}
                    required
                  />
                </div>
                <div>
                  <label className="text-sm font-medium">Daily Limit</label>
                  <Input
                    type="number"
                    value={formData.daily_limit}
                    onChange={(e) => setFormData({ ...formData, daily_limit: parseFloat(e.target.value) })}
                    required
                  />
                </div>
              </div>
              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={formData.is_sandbox}
                  onChange={(e) => setFormData({ ...formData, is_sandbox: e.target.checked })}
                  className="rounded"
                />
                <label className="text-sm">Sandbox mode</label>
              </div>
              <div className="flex gap-2">
                <Button type="submit" disabled={createMutation.isPending}>
                  {createMutation.isPending ? "Creating..." : "Create"}
                </Button>
                <Button type="button" variant="outline" onClick={() => setShowCreate(false)}>
                  Cancel
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      {(!Array.isArray(data) || data.length === 0) && !showCreate && (
        <EmptyState icon={CreditCard} title="No requisites" description="Add bank accounts for providers to receive deposits." />
      )}

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {Array.isArray(data) && data.map((req) => (
          <Card key={req.id}>
            <CardHeader className="flex flex-row items-start justify-between pb-2">
              <div>
                <CardTitle className="text-base">{req.bank_name}</CardTitle>
                <p className="text-sm text-muted-foreground">{req.holder_name}</p>
              </div>
              <Badge status={req.status}>{req.status}</Badge>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="text-sm">
                <p className="text-muted-foreground">Account</p>
                <p className="font-mono">{req.account_number}</p>
              </div>
              <div className="grid grid-cols-2 gap-2 text-sm">
                <div><p className="text-muted-foreground">Currency</p><p>{req.currency}</p></div>
                <div><p className="text-muted-foreground">Country</p><p>{req.country}</p></div>
              </div>
              <LimitProgress used={req.used_limit} limit={req.daily_limit} />
              <div className="flex justify-between text-xs text-muted-foreground">
                <span>Used: {formatCurrency(req.used_limit, req.currency)}</span>
                <span>Limit: {formatCurrency(req.daily_limit, req.currency)}</span>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
