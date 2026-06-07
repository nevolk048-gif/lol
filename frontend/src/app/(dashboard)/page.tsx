"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { motion } from "framer-motion";
import {
  DollarSign,
  ArrowLeftRight,
  Building2,
  CreditCard,
  TrendingUp,
  Clock,
  Activity,
} from "lucide-react";
import { api } from "@/services/api";
import { StatCard } from "@/components/widgets/stat-card";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { StatCardSkeleton } from "@/components/ui/skeleton";
import {
  TurnoverAreaChart,
  DailyBarChart,
  DistributionPieChart,
} from "@/components/charts/charts";
import { useWebSocket } from "@/hooks/use-websocket";
import { formatDate } from "@/lib/utils";
import { toast } from "sonner";

export default function DashboardPage() {
  const queryClient = useQueryClient();

  useWebSocket((event) => {
    if (event.type.startsWith("transaction")) {
      queryClient.invalidateQueries({ queryKey: ["dashboard"] });
      toast.info(`Transaction update: ${event.type}`);
    }
  });

  const { data, isLoading } = useQuery({
    queryKey: ["dashboard"],
    queryFn: () => api.getDashboard(),
    refetchInterval: 30000,
  });

  const stats = data?.stats;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Dashboard</h1>
        <p className="text-muted-foreground">Real-time overview of your payment operations</p>
      </div>

      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <StatCardSkeleton key={i} />
          ))}
        </div>
      ) : (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="grid gap-4 md:grid-cols-2 lg:grid-cols-4"
        >
          <StatCard title="Turnover (24h)" value={stats?.turnover_day ?? 0} format="currency" icon={DollarSign} trend={12.5} />
          <StatCard title="Turnover (Week)" value={stats?.turnover_week ?? 0} format="currency" icon={TrendingUp} trend={8.2} />
          <StatCard title="Turnover (Month)" value={stats?.turnover_month ?? 0} format="currency" icon={DollarSign} trend={15.7} />
          <StatCard title="Profit" value={stats?.profit ?? 0} format="currency" icon={TrendingUp} trend={5.3} />
          <StatCard title="Transactions" value={stats?.transaction_count ?? 0} format="number" icon={ArrowLeftRight} />
          <StatCard title="Active Providers" value={stats?.active_providers ?? 0} format="number" icon={Building2} />
          <StatCard title="Active Requisites" value={stats?.active_requisites ?? 0} format="number" icon={CreditCard} />
          <StatCard title="Conversion" value={stats?.conversion_rate ?? 0} format="percent" icon={Activity} />
        </motion.div>
      )}

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Clock className="h-4 w-4" />
              Hourly Turnover
            </CardTitle>
          </CardHeader>
          <CardContent>
            <TurnoverAreaChart data={data?.turnover_hourly ?? []} />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Daily Turnover</CardTitle>
          </CardHeader>
          <CardContent>
            <DailyBarChart data={data?.turnover_daily ?? []} />
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        <Card>
          <CardHeader><CardTitle>By Provider</CardTitle></CardHeader>
          <CardContent>
            <DistributionPieChart data={data?.by_provider ?? []} />
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>By Casino</CardTitle></CardHeader>
          <CardContent>
            <DistributionPieChart data={data?.by_casino ?? []} />
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>By Country</CardTitle></CardHeader>
          <CardContent>
            <DistributionPieChart data={data?.by_country ?? []} />
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader><CardTitle>Recent Events</CardTitle></CardHeader>
        <CardContent>
          <div className="space-y-3">
            {(data?.recent_events ?? []).map((event) => (
              <div key={event.id} className="flex items-center justify-between rounded-lg border border-border p-3">
                <div>
                  <p className="text-sm font-medium">{event.message}</p>
                  <p className="text-xs text-muted-foreground">{event.type}</p>
                </div>
                <span className="text-xs text-muted-foreground">{formatDate(event.timestamp)}</span>
              </div>
            ))}
            {(data?.recent_events ?? []).length === 0 && (
              <p className="text-center text-sm text-muted-foreground py-8">No recent events</p>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
