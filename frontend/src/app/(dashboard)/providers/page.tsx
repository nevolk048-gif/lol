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
import { StatCardSkeleton, EmptyState } from "@/components/ui/skeleton";
import { formatCurrency, formatNumber } from "@/lib/utils";
import { Building2, ArrowRight, Plus } from "lucide-react";
import { toast } from "sonner";
import { useI18n } from "@/hooks/use-i18n";
import type { Provider } from "@/types";

export default function ProvidersPage() {
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
    queryKey: ["providers"],
    queryFn: () => api.getProviders(),
  });

  const createMutation = useMutation({
    mutationFn: (data: typeof formData) => {
      const payload: Record<string, unknown> = { name: data.name, is_sandbox: data.is_sandbox };
      if (data.merchant_id) payload.merchant_id = data.merchant_id;
      if (data.base_url) payload.base_url = data.base_url;
      if (data.webhook_url) payload.webhook_url = data.webhook_url;
      if (data.secret_key) payload.secret_key = data.secret_key;
      return api.createProvider(payload);
    },
    onSuccess: () => {
      toast.success(t("providerCreated"));
      queryClient.invalidateQueries({ queryKey: ["providers"] });
      setShowCreate(false);
      setFormData({ name: "", merchant_id: "", base_url: "", webhook_url: "", secret_key: "", is_sandbox: false });
    },
    onError: () => toast.error(t("failedToCreate")),
  });

  if (isLoading) {
    return (
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => <StatCardSkeleton key={i} />)}
      </div>
    );
  }

  const renderCreateForm = () => (
    <Card>
      <CardHeader>
        <CardTitle>{t("createProvider")}</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={(e) => { e.preventDefault(); createMutation.mutate(formData); }} className="space-y-4">
          <div>
            <label className="text-sm font-medium">{t("name")} *</label>
            <Input
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              placeholder="MajorPay"
              required
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-sm font-medium">Merchant ID</label>
              <Input
                value={formData.merchant_id}
                onChange={(e) => setFormData({ ...formData, merchant_id: e.target.value })}
                placeholder="shop_c5cf9b51..."
              />
            </div>
            <div>
              <label className="text-sm font-medium">Base URL</label>
              <Input
                value={formData.base_url}
                onChange={(e) => setFormData({ ...formData, base_url: e.target.value })}
                placeholder="https://api.majorpay.io/api"
              />
            </div>
          </div>
          <div>
            <label className="text-sm font-medium">Secret Key</label>
            <Input
              type="password"
              value={formData.secret_key}
              onChange={(e) => setFormData({ ...formData, secret_key: e.target.value })}
              placeholder="sk_2d1c7f5807... (auto-generated if empty)"
            />
          </div>
          <div>
            <label className="text-sm font-medium">{t("webhookUrl")}</label>
            <Input
              value={formData.webhook_url}
              onChange={(e) => setFormData({ ...formData, webhook_url: e.target.value })}
              placeholder="https://your-domain.com/webhook"
            />
          </div>
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="sandbox"
              checked={formData.is_sandbox}
              onChange={(e) => setFormData({ ...formData, is_sandbox: e.target.checked })}
              className="rounded"
            />
            <label htmlFor="sandbox" className="text-sm">{t("sandboxMode")}</label>
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
            <h1 className="text-2xl font-bold">{t("providers")}</h1>
            <p className="text-muted-foreground">Payment provider network and performance</p>
          </div>
          <Button onClick={() => setShowCreate(true)}>
            <Plus className="h-4 w-4 mr-2" />
            {t("createProvider")}
          </Button>
        </div>
        {showCreate && renderCreateForm()}
        <EmptyState icon={Building2} title={t("noProviders")} description="Add your first payment provider to start routing." />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("providers")}</h1>
          <p className="text-muted-foreground">Payment provider network and performance</p>
        </div>
        <Button onClick={() => setShowCreate(true)}>
          <Plus className="h-4 w-4 mr-2" />
          {t("createProvider")}
        </Button>
      </div>

      {showCreate && renderCreateForm()}

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {(data as Provider[]).map((provider, i) => (
          <motion.div
            key={provider.id}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: i * 0.05 }}
          >
            <Link href={`/providers/${provider.id}`}>
              <Card className="hover:shadow-lg transition-all hover:border-primary/30 cursor-pointer group">
                <CardHeader className="flex flex-row items-start justify-between pb-2">
                  <div>
                    <CardTitle className="text-lg">{provider.name}</CardTitle>
                    <p className="text-xs text-muted-foreground font-mono mt-1">
                      {provider.api_key.slice(0, 16)}...
                    </p>
                  </div>
                  <Badge status={provider.status}>{t(provider.status)}</Badge>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div>
                      <p className="text-muted-foreground">Turnover</p>
                      <p className="font-semibold">{formatCurrency(provider.turnover ?? 0)}</p>
                    </div>
                    <div>
                      <p className="text-muted-foreground">{t("transactions")}</p>
                      <p className="font-semibold">{formatNumber(provider.transaction_count ?? 0)}</p>
                    </div>
                    <div>
                      <p className="text-muted-foreground">Avg Response</p>
                      <p className="font-semibold">{provider.avg_response_ms?.toFixed(0) ?? 0}ms</p>
                    </div>
                    <div>
                      <p className="text-muted-foreground">Environment</p>
                      <p className="font-semibold">{provider.is_sandbox ? t("sandbox") : "Production"}</p>
                    </div>
                  </div>
                  <div className="mt-4 flex items-center text-sm text-primary opacity-0 group-hover:opacity-100 transition-opacity">
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
