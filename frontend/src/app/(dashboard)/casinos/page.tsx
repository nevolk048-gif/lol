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
import { CreditCard, ArrowRight, Plus, Trash2 } from "lucide-react";
import { EmptyState, StatCardSkeleton } from "@/components/ui/skeleton";
import { toast } from "sonner";
import { useI18n } from "@/hooks/use-i18n";

export default function CasinosPage() {
  const { t } = useI18n();
  const [showCreate, setShowCreate] = useState(false);
  const [formData, setFormData] = useState({
    name: "",
    merchant_id: "",
    base_url: "",
    webhook_url: "",
    secret_key: "",
    is_sandbox: false,
  });

  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ["casinos"],
    queryFn: () => api.getCasinos(),
    select: (data) => Array.isArray(data) ? data : [],
  });

  const createMutation = useMutation({
    mutationFn: (data: typeof formData) => {
      const payload = {
        name: data.name,
        is_sandbox: data.is_sandbox,
        ...(data.merchant_id && { merchant_id: data.merchant_id }),
        ...(data.base_url && { base_url: data.base_url }),
        ...(data.webhook_url && { webhook_url: data.webhook_url }),
        ...(data.secret_key && { secret_key: data.secret_key }),
      };
      return api.createCasino(payload);
    },
    onSuccess: () => {
      toast.success(t("casinoCreated"));
      queryClient.invalidateQueries({ queryKey: ["casinos"] });
      setShowCreate(false);
      setFormData({ name: "", merchant_id: "", base_url: "", webhook_url: "", secret_key: "", is_sandbox: false });
    },
    onError: () => toast.error(t("failedToCreate")),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteCasino(id),
    onSuccess: () => {
      toast.success("Casino deleted successfully");
      queryClient.invalidateQueries({ queryKey: ["casinos"] });
    },
    onError: () => toast.error("Failed to delete casino"),
  });

  if (isLoading) {
    return (
      <div className="grid gap-4 md:grid-cols-3">
        {Array.from({ length: 3 }).map((_, i) => <StatCardSkeleton key={i} />)}
      </div>
    );
  }

  const renderCreateForm = () => (
    <Card>
      <CardHeader>
        <CardTitle>{t("createCasino")}</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={(e) => { e.preventDefault(); createMutation.mutate(formData); }} className="space-y-4">
          <div>
            <label className="text-sm font-medium">{t("name")} *</label>
            <Input
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              placeholder="Royal Casino"
              required
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-sm font-medium">Merchant ID</label>
              <Input
                value={formData.merchant_id}
                onChange={(e) => setFormData({ ...formData, merchant_id: e.target.value })}
                placeholder="casino_abc123..."
              />
            </div>
            <div>
              <label className="text-sm font-medium">Base URL</label>
              <Input
                value={formData.base_url}
                onChange={(e) => setFormData({ ...formData, base_url: e.target.value })}
                placeholder="https://api.casino.com"
              />
            </div>
          </div>
          <div>
            <label className="text-sm font-medium">Secret Key</label>
            <Input
              type="password"
              value={formData.secret_key}
              onChange={(e) => setFormData({ ...formData, secret_key: e.target.value })}
              placeholder="sk_casino_secret..."
            />
          </div>
          <div>
            <label className="text-sm font-medium">{t("webhookUrl")}</label>
            <Input
              value={formData.webhook_url}
              onChange={(e) => setFormData({ ...formData, webhook_url: e.target.value })}
              placeholder="https://casino.com/webhook"
            />
          </div>
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="sandbox-casino"
              checked={formData.is_sandbox}
              onChange={(e) => setFormData({ ...formData, is_sandbox: e.target.checked })}
              className="rounded"
            />
            <label htmlFor="sandbox-casino" className="text-sm">{t("sandboxMode")}</label>
          </div>
          <div className="flex gap-2">
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending ? t("creating") : t("create")}
            </Button>
            <Button type="button" variant="outline" onClick={() => setShowCreate(false)}>
              {t("cancel")}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );

  if (!Array.isArray(data) || data.length === 0) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">{t("casinos")}</h1>
            <p className="text-muted-foreground">Connected casino partners and their performance</p>
          </div>
          <Button onClick={() => setShowCreate(true)}>
            <Plus className="h-4 w-4 mr-2" />
            {t("createCasino")}
          </Button>
        </div>
        {showCreate && renderCreateForm()}
        <EmptyState icon={CreditCard} title={t("noCasinos")} description="Connect your first casino partner." />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("casinos")}</h1>
          <p className="text-muted-foreground">Connected casino partners and their performance</p>
        </div>
        <Button onClick={() => setShowCreate(true)}>
          <Plus className="h-4 w-4 mr-2" />
          {t("createCasino")}
        </Button>
      </div>

      {showCreate && renderCreateForm()}

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {data.map((casino, i) => (
          <motion.div key={casino.id} initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.05 }}>
            <Card className="hover:shadow-lg transition-all hover:border-primary/30 group relative">
              <div className="absolute top-2 right-2 z-10">
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-8 w-8 p-0 text-destructive hover:text-destructive hover:bg-destructive/10"
                  onClick={(e) => {
                    e.preventDefault();
                    if (confirm(`Delete ${casino.name}?`)) {
                      deleteMutation.mutate(casino.id);
                    }
                  }}
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
              <Link href={`/casinos/${casino.id}`}>
                <CardHeader className="flex flex-row items-start justify-between pr-12">
                  <CardTitle>{casino.name}</CardTitle>
                  <Badge status={casino.status}>{t(casino.status)}</Badge>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 gap-4 text-sm mb-4">
                    <div>
                      <p className="text-muted-foreground">Turnover</p>
                      <p className="font-semibold">{formatCurrency(casino.turnover ?? 0)}</p>
                    </div>
                    <div>
                      <p className="text-muted-foreground">{t("transactions")}</p>
                      <p className="font-semibold">{formatNumber(casino.transaction_count ?? 0)}</p>
                    </div>
                  </div>
                  <p className="text-xs font-mono text-muted-foreground">{casino.api_key.slice(0, 20)}...</p>
                  <div className="mt-3 flex items-center text-sm text-primary opacity-0 group-hover:opacity-100 transition-opacity">
                    View details <ArrowRight className="ml-1 h-4 w-4" />
                  </div>
                </CardContent>
              </Link>
            </Card>
          </motion.div>
        ))}
      </div>
    </div>
  );
}
