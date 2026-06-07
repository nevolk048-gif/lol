"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/services/api";
import { StatCard } from "@/components/widgets/stat-card";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { DailyBarChart, DistributionPieChart } from "@/components/charts/charts";
import { StatCardSkeleton } from "@/components/ui/skeleton";
import { DollarSign, TrendingUp, Percent, Wallet } from "lucide-react";

export default function FinancePage() {
  const { data, isLoading } = useQuery({
    queryKey: ["finance"],
    queryFn: () => api.getFinance(),
  });

  if (isLoading) {
    return (
      <div className="grid gap-4 md:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => <StatCardSkeleton key={i} />)}
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Finance</h1>
        <p className="text-muted-foreground">Revenue, commissions, and payout analytics</p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <StatCard title="Total Turnover" value={data?.turnover ?? 0} format="currency" icon={DollarSign} />
        <StatCard title="Profit" value={data?.profit ?? 0} format="currency" icon={TrendingUp} trend={8.4} />
        <StatCard title="Commissions" value={data?.commissions ?? 0} format="currency" icon={Percent} />
        <StatCard title="Payouts" value={data?.payouts ?? 0} format="currency" icon={Wallet} />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader><CardTitle>Profit by Day</CardTitle></CardHeader>
          <CardContent>
            <DailyBarChart data={data?.profit_daily ?? []} />
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>Profit by Casino</CardTitle></CardHeader>
          <CardContent>
            <DistributionPieChart data={data?.profit_by_casino ?? []} />
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
