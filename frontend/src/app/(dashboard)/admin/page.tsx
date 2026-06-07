"use client";

import { useQuery } from "@tanstack/react-query";
import { type ColumnDef } from "@tanstack/react-table";
import { api } from "@/services/api";
import { DataTable } from "@/components/widgets/data-table";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatDate } from "@/lib/utils";
import type { User, AuditLog } from "@/types";

export default function AdminPage() {
  const { data: users } = useQuery({
    queryKey: ["users"],
    queryFn: () => api.getUsers(),
  });

  const { data: auditLogs } = useQuery({
    queryKey: ["audit-logs"],
    queryFn: () => api.getAuditLogs({ per_page: "20" }),
  });

  const userColumns: ColumnDef<User>[] = [
    { accessorKey: "email", header: "Email" },
    { accessorKey: "role", header: "Role" },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <Badge status={row.original.status}>{row.original.status}</Badge>,
    },
    {
      accessorKey: "created_at",
      header: "Created",
      cell: ({ row }) => formatDate(row.original.created_at),
    },
  ];

  const logColumns: ColumnDef<AuditLog>[] = [
    { accessorKey: "action", header: "Action" },
    { accessorKey: "entity_type", header: "Entity" },
    { accessorKey: "user_email", header: "User" },
    { accessorKey: "ip_address", header: "IP" },
    {
      accessorKey: "created_at",
      header: "Time",
      cell: ({ row }) => formatDate(row.original.created_at),
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Admin Panel</h1>
        <p className="text-muted-foreground">User management and audit logs</p>
      </div>

      <Card>
        <CardHeader><CardTitle>Users</CardTitle></CardHeader>
        <CardContent>
          <DataTable data={users ?? []} columns={userColumns} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle>Audit Logs</CardTitle></CardHeader>
        <CardContent>
          <DataTable data={auditLogs ?? []} columns={logColumns} />
        </CardContent>
      </Card>
    </div>
  );
}
