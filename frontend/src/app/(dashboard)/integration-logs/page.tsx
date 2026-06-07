"use client";

import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { type ColumnDef } from "@tanstack/react-table";
import { api } from "@/services/api";
import { DataTable } from "@/components/widgets/data-table";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { formatDate, truncate } from "@/lib/utils";
import type { IntegrationLog } from "@/types";
import { Search } from "lucide-react";

export default function IntegrationLogsPage() {
  const [endpoint, setEndpoint] = useState("");
  const [method, setMethod] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["integration-logs", endpoint, method],
    queryFn: () =>
      api.getIntegrationLogs({
        ...(endpoint && { endpoint }),
        ...(method && { method }),
        per_page: "50",
      }),
  });

  const columns: ColumnDef<IntegrationLog>[] = [
    {
      accessorKey: "created_at",
      header: "Time",
      cell: ({ row }) => formatDate(row.original.created_at),
    },
    { accessorKey: "method", header: "Method" },
    { accessorKey: "endpoint", header: "Endpoint" },
    {
      accessorKey: "status_code",
      header: "Status",
      cell: ({ row }) => {
        const code = row.original.status_code;
        const variant = code >= 400 ? "CANCELLED" : code >= 300 ? "WAITING_PAYMENT" : "PAID";
        return <Badge status={variant}>{code}</Badge>;
      },
    },
    {
      accessorKey: "duration_ms",
      header: "Duration",
      cell: ({ row }) => `${row.original.duration_ms}ms`,
    },
    {
      accessorKey: "transaction_id",
      header: "Transaction",
      cell: ({ row }) =>
        row.original.transaction_id ? (
          <span className="font-mono text-xs">{truncate(row.original.transaction_id, 10)}</span>
        ) : "—",
    },
    {
      accessorKey: "is_sandbox",
      header: "Env",
      cell: ({ row }) => (row.original.is_sandbox ? "Sandbox" : "Prod"),
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Integration Logs</h1>
        <p className="text-muted-foreground">API request logs for casinos and providers</p>
      </div>

      <div className="flex flex-wrap gap-3">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search endpoint..."
            className="pl-9 w-64"
            value={endpoint}
            onChange={(e) => setEndpoint(e.target.value)}
          />
        </div>
        <select
          className="h-10 rounded-lg border border-border bg-background px-3 text-sm"
          value={method}
          onChange={(e) => setMethod(e.target.value)}
        >
          <option value="">All Methods</option>
          <option value="GET">GET</option>
          <option value="POST">POST</option>
          <option value="PUT">PUT</option>
          <option value="DELETE">DELETE</option>
        </select>
        <Button variant="outline" onClick={() => { setEndpoint(""); setMethod(""); }}>
          Clear
        </Button>
      </div>

      {!isLoading && <DataTable data={data ?? []} columns={columns} />}
    </div>
  );
}
