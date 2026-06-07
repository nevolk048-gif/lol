"use client";

import { useQuery } from "@tanstack/react-query";
import { type ColumnDef } from "@tanstack/react-table";
import { useState } from "react";
import { api } from "@/services/api";
import { DataTable } from "@/components/widgets/data-table";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { TableSkeleton } from "@/components/ui/skeleton";
import { formatCurrency, formatDate, truncate } from "@/lib/utils";
import type { Transaction, TransactionStatus } from "@/types";
import { Filter } from "lucide-react";

const statuses: TransactionStatus[] = [
  "NEW", "ASSIGNED", "WAITING_PAYMENT", "PAID", "EXPIRED", "CANCELLED",
];

export default function TransactionsPage() {
  const [statusFilter, setStatusFilter] = useState("");
  const [countryFilter, setCountryFilter] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["transactions", statusFilter, countryFilter],
    queryFn: () =>
      api.getTransactions({
        ...(statusFilter && { status: statusFilter }),
        ...(countryFilter && { country: countryFilter }),
        per_page: "50",
      }),
  });

  const columns: ColumnDef<Transaction>[] = [
    {
      accessorKey: "id",
      header: "ID",
      cell: ({ row }) => (
        <span className="font-mono text-xs">{truncate(row.original.id, 12)}</span>
      ),
    },
    { accessorKey: "casino_name", header: "Casino" },
    { accessorKey: "provider_name", header: "Provider" },
    { accessorKey: "requisite_bank", header: "Requisite" },
    { accessorKey: "country", header: "Country" },
    {
      accessorKey: "amount",
      header: "Amount",
      cell: ({ row }) => formatCurrency(row.original.amount, row.original.currency),
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <Badge status={row.original.status}>{row.original.status.replace("_", " ")}</Badge>,
    },
    {
      accessorKey: "created_at",
      header: "Date",
      cell: ({ row }) => formatDate(row.original.created_at),
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Transactions</h1>
          <p className="text-muted-foreground">All deposit requests and their routing status</p>
        </div>
      </div>

      <div className="flex flex-wrap gap-3">
        <div className="flex items-center gap-2">
          <Filter className="h-4 w-4 text-muted-foreground" />
          <select
            className="h-10 rounded-lg border border-border bg-background px-3 text-sm"
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
          >
            <option value="">All Statuses</option>
            {statuses.map((s) => (
              <option key={s} value={s}>{s.replace("_", " ")}</option>
            ))}
          </select>
        </div>
        <Input
          placeholder="Filter by country (US, DE...)"
          className="w-48"
          value={countryFilter}
          onChange={(e) => setCountryFilter(e.target.value.toUpperCase())}
        />
        <Button variant="outline" onClick={() => { setStatusFilter(""); setCountryFilter(""); }}>
          Clear filters
        </Button>
      </div>

      {isLoading ? <TableSkeleton /> : <DataTable data={data ?? []} columns={columns} />}
    </div>
  );
}
