"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { motion } from "framer-motion";
import { api } from "@/services/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { StatCardSkeleton, EmptyState } from "@/components/ui/skeleton";
import { formatCurrency, formatNumber } from "@/lib/utils";
import { Building2, ArrowRight } from "lucide-react";
import type { Provider } from "@/types";

export default function ProvidersPage() {
  const { data, isLoading } = useQuery({
    queryKey: ["providers"],
    queryFn: () => api.getProviders(),
  });

  if (isLoading) {
    return (
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => <StatCardSkeleton key={i} />)}
      </div>
    );
  }

  if (!data?.length) {
    return <EmptyState icon={Building2} title="No providers" description="Add your first payment provider to start routing." />;
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Providers</h1>
        <p className="text-muted-foreground">Payment provider network and performance</p>
      </div>

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
                  <Badge status={provider.status}>{provider.status}</Badge>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div>
                      <p className="text-muted-foreground">Turnover</p>
                      <p className="font-semibold">{formatCurrency(provider.turnover ?? 0)}</p>
                    </div>
                    <div>
                      <p className="text-muted-foreground">Transactions</p>
                      <p className="font-semibold">{formatNumber(provider.transaction_count ?? 0)}</p>
                    </div>
                    <div>
                      <p className="text-muted-foreground">Avg Response</p>
                      <p className="font-semibold">{provider.avg_response_ms?.toFixed(0) ?? 0}ms</p>
                    </div>
                    <div>
                      <p className="text-muted-foreground">Environment</p>
                      <p className="font-semibold">{provider.is_sandbox ? "Sandbox" : "Production"}</p>
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
