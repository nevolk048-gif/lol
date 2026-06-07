"use client";

import { useQuery } from "@tanstack/react-query";
import { motion } from "framer-motion";
import { api } from "@/services/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ArrowRight, Zap, Building2, CreditCard } from "lucide-react";

export default function RoutingPage() {
  const { data: rules } = useQuery({
    queryKey: ["route-rules"],
    queryFn: () => api.getRouteRules(),
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Routing Engine</h1>
        <p className="text-muted-foreground">Automatic deposit routing configuration</p>
      </div>

      <Card className="overflow-hidden">
        <CardContent className="p-8">
          <div className="flex items-center justify-center gap-4 flex-wrap">
            {[
              { icon: CreditCard, label: "Casino", color: "bg-blue-500/10 text-blue-500" },
              { icon: ArrowRight, label: "", color: "text-muted-foreground" },
              { icon: Zap, label: "Aggregator", color: "bg-primary/10 text-primary" },
              { icon: ArrowRight, label: "", color: "text-muted-foreground" },
              { icon: Building2, label: "Provider", color: "bg-cyan-500/10 text-cyan-500" },
              { icon: ArrowRight, label: "", color: "text-muted-foreground" },
              { icon: CreditCard, label: "Requisite", color: "bg-emerald-500/10 text-emerald-500" },
            ].map((step, i) => (
              <motion.div
                key={i}
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ delay: i * 0.1 }}
                className="flex flex-col items-center gap-2"
              >
                {step.label ? (
                  <div className={`rounded-xl p-4 ${step.color}`}>
                    <step.icon className="h-8 w-8" />
                  </div>
                ) : (
                  <step.icon className={`h-6 w-6 ${step.color}`} />
                )}
                {step.label && <span className="text-sm font-medium">{step.label}</span>}
              </motion.div>
            ))}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle>Route Rules</CardTitle></CardHeader>
        <CardContent>
          <div className="space-y-3">
            {rules?.map((rule) => (
              <div key={rule.id} className="flex items-center justify-between rounded-lg border border-border p-4">
                <div className="flex items-center gap-4">
                  <div className="text-center">
                    <p className="text-xs text-muted-foreground">Priority</p>
                    <p className="text-lg font-bold">{rule.priority}</p>
                  </div>
                  <div>
                    <p className="font-medium">{rule.provider_name || rule.provider_id}</p>
                    <p className="text-sm text-muted-foreground">
                      {rule.country || "Any country"} · {rule.currency || "Any currency"} · Weight: {rule.weight}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  {rule.is_fallback && <Badge className="bg-warning/10 text-warning border-warning/20">Fallback</Badge>}
                  <Badge status={rule.status}>{rule.status}</Badge>
                </div>
              </div>
            ))}
            {(!rules || rules.length === 0) && (
              <p className="text-center text-muted-foreground py-8">No routing rules configured</p>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
