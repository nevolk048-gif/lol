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
import { useI18n } from "@/hooks/use-i18n";
import type { User, AuditLog } from "@/types";

export default function AdminPage() {
  const { t } = useI18n();
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
      toast.success(t("userCreated"));
      queryClient.invalidateQueries({ queryKey: ["users"] });
      setShowCreate(false);
      setFormData({ email: "", password: "", role: "ANALYST" });
    },
    onError: () => toast.error(t("failedToCreate")),
  });

  const userColumns: ColumnDef<User>[] = [
    { accessorKey: "email", header: t("email") },
    { accessorKey: "role", header: t("role") },
    {
      accessorKey: "status",
      header: t("status"),
      cell: ({ row }) => <Badge status={row.original.status}>{t(row.original.status)}</Badge>,
    },
    {
      accessorKey: "created_at",
      header: t("created"),
      cell: ({ row }) => formatDate(row.original.created_at),
    },
  ];

  const logColumns: ColumnDef<AuditLog>[] = [
    { accessorKey: "action", header: t("action") },
    { accessorKey: "entity_type", header: t("entity") },
    { accessorKey: "user_email", header: t("user") },
    { accessorKey: "ip_address", header: t("ipAddress") },
    {
      accessorKey: "created_at",
      header: t("time"),
      cell: ({ row }) => formatDate(row.original.created_at),
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("adminPanel")}</h1>
          <p className="text-muted-foreground">{t("userManagement")}</p>
        </div>
        <Button onClick={() => setShowCreate(true)}>
          <Plus className="h-4 w-4 mr-2" />
          {t("createUser")}
        </Button>
      </div>

      {showCreate && (
        <Card>
          <CardHeader>
            <CardTitle>{t("createNewUser")}</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={(e) => { e.preventDefault(); createMutation.mutate(formData); }} className="space-y-4">
              <div>
                <label className="text-sm font-medium">{t("email")} / {t("username")}</label>
                <Input
                  value={formData.email}
                  onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                  placeholder="user@example.com"
                  required
                />
              </div>
              <div>
                <label className="text-sm font-medium">{t("password")}</label>
                <Input
                  type="password"
                  value={formData.password}
                  onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                  placeholder={t("minCharacters").replace("{count}", "6")}
                  minLength={6}
                  required
                />
              </div>
              <div>
                <label className="text-sm font-medium">{t("role")}</label>
                <select
                  className="w-full h-10 rounded-lg border border-border bg-background px-3 text-sm"
                  value={formData.role}
                  onChange={(e) => setFormData({ ...formData, role: e.target.value as any })}
                >
                  <option value="ANALYST">{t("roleAnalyst")}</option>
                  <option value="SUPPORT">{t("roleSupport")}</option>
                  <option value="ADMIN">{t("roleAdmin")}</option>
                  <option value="SUPER_ADMIN">{t("roleSuperAdmin")}</option>
                </select>
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
      )}

      <Card>
        <CardHeader><CardTitle>{t("users")}</CardTitle></CardHeader>
        <CardContent>
          <DataTable data={users ?? []} columns={userColumns} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle>{t("auditLogs")}</CardTitle></CardHeader>
        <CardContent>
          <DataTable data={auditLogs ?? []} columns={logColumns} />
        </CardContent>
      </Card>
    </div>
  );
}
