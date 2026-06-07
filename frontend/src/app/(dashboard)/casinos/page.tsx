"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { motion } from "framer-motion";
import { api } from "@/services/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { formatCurrency, formatNumber } from "@/lib/utils";
import { CreditCard, ArrowRight, Plus } from "lucide-react";
import { EmptyState, StatCardSkeleton } from "@/components/ui/skeleton";
import { toast } from "sonner";

export default function CasinosPage() {
  const [showCreate, setShowCreate] = useState(false);
  const [formData, setFormData] = useState({
    name: "",
    webhook_url: "",
    is_sandbox: false,
  });

  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ["casinos"],
    queryFn: () => api.getCasinos(),
  });

  const createMutation = useMutation({
    mutationFn: (data: typeof formData) => api.createCasino(data),
    onSuccess: () => {
      toast.success("Casino created successfully");
      queryClient.invalidateQueries({ queryKey: ["casinos"] });
      setShowCreate(false);
      setFormData({ name: "", webhook_url: "", is_sandbox: false });
    },
    onError: () => toast.error("Failed to create casino"),
  });

  if (isLoading) {
    return (
      <div className="grid gap-4 md:grid-cols-3">
        {Array.from({ length: 3 }).map((_, i) => <StatCardSkeleton key={i} />)}
      </div>
    );
  }

  if (!Array.isArray(data) || data.length === 0) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Casinos</h1>
            <p className="text-muted-foreground">Connected casino partners and their performance</p>
          </div>
          <Button onClick={() => setShowCreate(true)}>
            <Plus className="h-4 w-4 mr-2" />
            Create Casino
          </Button>
        </div>
        {showCreate && (
          <Card>
            <CardHeader>
              <CardTitle>Create New Casino</CardTitle>
            </CardHeader>
            <CardContent>
              <form onSubmit={(e) => { e.preventDefault(); createMutation.mutate(formData); }} className="space-y-4">
                <div>
                  <label className="text-sm font-medium">Name</label>
                  <Input
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="Casino name"
                    required
                  />
                </div>
                <div>
                  <label className="text-sm font-medium">Webhook URL (optional)</label>
                  <Input
                    value={formData.webhook_url}
                    onChange={(e) => setFormData({ ...formData, webhook_url: e.target.value })}
                    placeholder="https://casino.com/webhook"
                  />
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
        <EmptyState icon={CreditCard} title="No casinos" description="Connect your first casino partner." />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Casinos</h1>
          <p className="text-muted-foreground">Connected casino partners and their performance</p>
        </div>
        <Button onClick={() => setShowCreate(true)}>
          <Plus className="h-4 w-4 mr-2" />
          Create Casino
        </Button>
      </div>

      {showCreate && (
        <Card>
          <CardHeader>
            <CardTitle>Create New Casino</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={(e) => { e.preventDefault(); createMutation.mutate(formData); }} className="space-y-4">
              <div>
                <label className="text-sm font-medium">Name</label>
                <Input
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="Casino name"
                  required
                />
              </div>
              <div>
                <label className="text-sm font-medium">Webhook URL (optional)</label>
                <Input
                  value={formData.webhook_url}
                  onChange={(e) => setFormData({ ...formData, webhook_url: e.target.value })}
                  placeholder="https://casino.com/webhook"
                />
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

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {data.map((casino, i) => (
          <motion.div key={casino.id} initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.05 }}>
            <Link href={`/casinos/${casino.id}`}>
              <Card className="hover:shadow-lg transition-all hover:border-primary/30 cursor-pointer group">
                <CardHeader className="flex flex-row items-start justify-between">
                  <CardTitle>{casino.name}</CardTitle>
                  <Badge status={casino.status}>{casino.status}</Badge>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 gap-4 text-sm mb-4">
                    <div>
                      <p className="text-muted-foreground">Turnover</p>
                      <p className="font-semibold">{formatCurrency(casino.turnover ?? 0)}</p>
                    </div>
                    <div>
                      <p className="text-muted-foreground">Transactions</p>
                      <p className="font-semibold">{formatNumber(casino.transaction_count ?? 0)}</p>
                    </div>
                  </div>
                  <p className="text-xs font-mono text-muted-foreground">{casino.api_key.slice(0, 20)}...</p>
                  <div className="mt-3 flex items-center text-sm text-primary opacity-0 group-hover:opacity-100 transition-opacity">
                    View details <ArrowRight className="ml-1 h-4 w-4" />
                  </div>
                </CardContent>
              </Card>
            </Link>
          </motion.div>
        ))}
      </div>
    </div>
  );
}
