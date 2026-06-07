"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { type ColumnDef } from "@tanstack/react-table";
import { api } from "@/services/api";
import { DataTable } from "@/components/widgets/data-table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatDate } from "@/lib/utils";
import { Plus } from "lucide-react";
import { toast } from "sonner";
import type { User, AuditLog } from "@/types";

export default function AdminPage() {
  const [showCreate, setShowCreate] = useState(false);
  const [formData, setFormData] = useState({
    email: "",
    password: "",
    role: "ANALYST" as "SUPER_ADMIN" | "ADMIN" | "SUPPORT" | "ANALYST",
  });

  const queryClient = useQueryClient();

  const { data: users } = useQuery({
    queryKey: ["users"],
    queryFn: () => api.getUsers(),
  });

  const { data: auditLogs } = useQuery({
    queryKey: ["audit-logs"],
    queryFn: () => api.getAuditLogs({ per_page: "20" }),
  });

  const createMutation = useMutation({
    mutationFn: (data: typeof formData) => api.createUser(data),
    onSuccess: () => {
      toast.success("User created successfully");
      queryClient.invalidateQueries({ queryKey: ["users"] });
      setShowCreate(false);
      setFormData({ email: "", password: "", role: "ANALYST" });
    },
    onError: () => toast.error("Failed to create user"),
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
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Admin Panel</h1>
          <p className="text-muted-foreground">User management and audit logs</p>
        </div>
        <Button onClick={() => setShowCreate(true)}>
          <Plus className="h-4 w-4 mr-2" />
          Create User
        </Button>
      </div>

      {showCreate && (
        <Card>
          <CardHeader>
            <CardTitle>Create New User</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={(e) => { e.preventDefault(); createMutation.mutate(formData); }} className="space-y-4">
              <div>
                <label className="text-sm font-medium">Email / Username</label>
                <Input
                  value={formData.email}
                  onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                  placeholder="user@example.com"
                  required
                />
              </div>
              <div>
                <label className="text-sm font-medium">Password</label>
                <Input
                  type="password"
                  value={formData.password}
                  onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                  placeholder="Min 6 characters"
                  minLength={6}
                  required
                />
              </div>
              <div>
                <label className="text-sm font-medium">Role</label>
                <select
                  className="w-full h-10 rounded-lg border border-border bg-background px-3 text-sm"
                  value={formData.role}
                  onChange={(e) => setFormData({ ...formData, role: e.target.value as any })}
                >
                  <option value="ANALYST">ANALYST - View only</option>
                  <option value="SUPPORT">SUPPORT - View + Logs</option>
                  <option value="ADMIN">ADMIN - Full access (no user mgmt)</option>
                  <option value="SUPER_ADMIN">SUPER_ADMIN - Full access</option>
                </select>
              </div>
              <div className="flex gap-2">
                <Button type="submit" disabled={createMutation.isPending}>
                  {createMutation.isPending ? "Creating..." : "Create"}
                </Button>
                <Button type="button" variant="outline" onClick={() => setShowCreate(false)}>
                  Cancel
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

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
