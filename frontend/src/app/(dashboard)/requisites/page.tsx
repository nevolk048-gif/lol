"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/services/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { LimitProgress } from "@/components/ui/progress";
import { TableSkeleton, EmptyState } from "@/components/ui/skeleton";
import { formatCurrency } from "@/lib/utils";
import { CreditCard } from "lucide-react";

export default function RequisitesPage() {
  const { data, isLoading } = useQuery({
    queryKey: ["requisites"],
    queryFn: () => api.getRequisites(),
  });

  if (isLoading) return <TableSkeleton rows={6} />;

  if (!data?.length) {
    return <EmptyState icon={CreditCard} title="No requisites" description="Add bank accounts for providers to receive deposits." />;
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Requisites</h1>
        <p className="text-muted-foreground">Bank accounts and daily limit management</p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {data.map((req) => (
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
