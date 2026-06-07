"use client";

import * as Progress from "@radix-ui/react-progress";
import { cn } from "@/lib/utils";

interface LimitProgressProps {
  used: number;
  limit: number;
  className?: string;
}

export function LimitProgress({ used, limit, className }: LimitProgressProps) {
  const percentage = limit > 0 ? Math.min((used / limit) * 100, 100) : 0;
  const available = Math.max(limit - used, 0);

  const colorClass =
    percentage >= 90
      ? "bg-destructive"
      : percentage >= 70
        ? "bg-warning"
        : "bg-primary";

  return (
    <div className={cn("space-y-2", className)}>
      <div className="flex justify-between text-xs text-muted-foreground">
        <span>{percentage.toFixed(0)}% used</span>
        <span>{available.toLocaleString()} available</span>
      </div>
      <Progress.Root
        className="relative h-2 w-full overflow-hidden rounded-full bg-secondary"
        value={percentage}
      >
        <Progress.Indicator
          className={cn("h-full rounded-full transition-all duration-500", colorClass)}
          style={{ width: `${percentage}%` }}
        />
      </Progress.Root>
    </div>
  );
}
