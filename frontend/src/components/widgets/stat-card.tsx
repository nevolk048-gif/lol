"use client";

import { motion } from "framer-motion";
import { cn, formatCurrency, formatNumber, formatPercent } from "@/lib/utils";
import { TrendingUp, TrendingDown } from "lucide-react";

interface StatCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  trend?: number;
  icon?: React.ComponentType<{ className?: string }>;
  className?: string;
  format?: "currency" | "number" | "percent" | "raw";
}

export function StatCard({
  title,
  value,
  subtitle,
  trend,
  icon: Icon,
  className,
  format = "raw",
}: StatCardProps) {
  const formatted =
    format === "currency"
      ? formatCurrency(Number(value))
      : format === "number"
        ? formatNumber(Number(value))
        : format === "percent"
          ? formatPercent(Number(value))
          : value;

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className={cn(
        "rounded-xl border border-border bg-card p-6 shadow-sm hover:shadow-md transition-shadow",
        className
      )}
    >
      <div className="flex items-start justify-between">
        <div className="space-y-2">
          <p className="text-sm font-medium text-muted-foreground">{title}</p>
          <p className="text-2xl font-bold tracking-tight">{formatted}</p>
          {subtitle && (
            <p className="text-xs text-muted-foreground">{subtitle}</p>
          )}
        </div>
        {Icon && (
          <div className="rounded-lg bg-primary/10 p-2.5">
            <Icon className="h-5 w-5 text-primary" />
          </div>
        )}
      </div>
      {trend !== undefined && (
        <div className="mt-3 flex items-center gap-1 text-xs">
          {trend >= 0 ? (
            <TrendingUp className="h-3 w-3 text-success" />
          ) : (
            <TrendingDown className="h-3 w-3 text-destructive" />
          )}
          <span className={trend >= 0 ? "text-success" : "text-destructive"}>
            {Math.abs(trend).toFixed(1)}%
          </span>
          <span className="text-muted-foreground">vs last period</span>
        </div>
      )}
    </motion.div>
  );
}
