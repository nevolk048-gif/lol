"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { motion } from "framer-motion";
import { api } from "@/services/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { formatCurrency, formatNumber } from "@/lib/utils";
import { CreditCard, ArrowRight } from "lucide-react";
import { EmptyState, StatCardSkeleton } from "@/components/ui/skeleton";

export default function CasinosPage() {
  const { data, isLoading } = useQuery({
    queryKey: ["casinos"],
    queryFn: () => api.getCasinos(),
  });

  if (isLoading) {
    return (
      <div className="grid gap-4 md:grid-cols-3">
        {Array.from({ length: 3 }).map((_, i) => <StatCardSkeleton key={i} />)}
      </div>
    );
  }

  if (!data?.length) {
    return <EmptyState icon={CreditCard} title="No casinos" description="Connect your first casino partner." />;
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Casinos</h1>
        <p className="text-muted-foreground">Connected casino partners and their performance</p>
      </div>

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
