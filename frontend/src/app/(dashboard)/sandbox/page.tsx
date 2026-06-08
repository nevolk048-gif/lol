"use client";

import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { api } from "@/services/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { FlaskConical, Zap, Play, BarChart3 } from "lucide-react";
import { toast } from "sonner";

export default function SandboxPage() {
  const [amount, setAmount] = useState("5000");
  const [trafficCount, setTrafficCount] = useState("10");
  const [casinoId, setCasinoId] = useState("");

  const { data: casinos } = useQuery({
    queryKey: ["casinos"],
    queryFn: () => api.getCasinos(),
  });

  const sandboxCasinos = Array.isArray(casinos) ? casinos.filter((c) => c.is_sandbox) : [];

  const setupMutation = useMutation({
    mutationFn: () => api.sandboxSetup(),
    onSuccess: (data) => {
      toast.success("Sandbox environment created");
      const d = data as { casino?: { id: string } };
      if (d.casino?.id) setCasinoId(d.casino.id);
    },
    onError: () => toast.error("Setup failed"),
  });

  const depositMutation = useMutation({
    mutationFn: () => api.sandboxDeposit(casinoId, parseFloat(amount)),
    onSuccess: () => toast.success("Test deposit created"),
    onError: () => toast.error("Deposit failed"),
  });

  const trafficMutation = useMutation({
    mutationFn: () => api.sandboxGenerateTraffic(casinoId, parseInt(trafficCount)),
    onSuccess: (data) => {
      const d = data as { count?: number };
      toast.success(`Generated ${d.count ?? 0} transactions`);
    },
    onError: () => toast.error("Traffic generation failed"),
  });

  const statsMutation = useMutation({
    mutationFn: () => api.sandboxGenerateStats(),
    onSuccess: (data) => toast.success(data.message),
    onError: () => toast.error("Stats generation failed"),
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <div className="rounded-lg bg-warning/10 p-2">
          <FlaskConical className="h-6 w-6 text-warning" />
        </div>
        <div>
          <h1 className="text-2xl font-bold">Sandbox Mode</h1>
          <p className="text-muted-foreground">Test environment with simulated data</p>
        </div>
        <Badge className="ml-auto bg-warning/10 text-warning border-warning/20">SANDBOX</Badge>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2"><Zap className="h-4 w-4" /> Setup</CardTitle>
            <CardDescription>Create sandbox casino, provider, and routing rules</CardDescription>
          </CardHeader>
          <CardContent>
            <Button onClick={() => setupMutation.mutate()} disabled={setupMutation.isPending}>
              {setupMutation.isPending ? "Setting up..." : "Initialize Sandbox"}
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2"><Play className="h-4 w-4" /> Test Deposit</CardTitle>
            <CardDescription>Create a single test deposit request</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <select
              className="w-full h-10 rounded-lg border border-border bg-background px-3 text-sm"
              value={casinoId}
              onChange={(e) => setCasinoId(e.target.value)}
            >
              <option value="">Select sandbox casino</option>
              {sandboxCasinos.map((c) => (
                <option key={c.id} value={c.id}>{c.name}</option>
              ))}
            </select>
            <Input type="number" value={amount} onChange={(e) => setAmount(e.target.value)} placeholder="Amount" />
            <Button onClick={() => depositMutation.mutate()} disabled={!casinoId || depositMutation.isPending}>
              Create Test Deposit
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Generate Traffic</CardTitle>
            <CardDescription>Simulate multiple deposit requests</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <Input type="number" value={trafficCount} onChange={(e) => setTrafficCount(e.target.value)} placeholder="Count (1-100)" />
            <Button onClick={() => trafficMutation.mutate()} disabled={!casinoId || trafficMutation.isPending} variant="secondary">
              Generate Traffic
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2"><BarChart3 className="h-4 w-4" /> Generate Stats</CardTitle>
            <CardDescription>Populate dashboard with sandbox analytics data</CardDescription>
          </CardHeader>
          <CardContent>
            <Button onClick={() => statsMutation.mutate()} disabled={statsMutation.isPending} variant="outline">
              Generate Statistics
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
