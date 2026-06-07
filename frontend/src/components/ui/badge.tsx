import { cn } from "@/lib/utils";

const statusColors: Record<string, string> = {
  NEW: "bg-blue-500/10 text-blue-500 border-blue-500/20",
  ASSIGNED: "bg-indigo-500/10 text-indigo-500 border-indigo-500/20",
  WAITING_PAYMENT: "bg-warning/10 text-warning border-warning/20",
  PAID: "bg-success/10 text-success border-success/20",
  EXPIRED: "bg-muted text-muted-foreground border-border",
  CANCELLED: "bg-destructive/10 text-destructive border-destructive/20",
  ACTIVE: "bg-success/10 text-success border-success/20",
  INACTIVE: "bg-muted text-muted-foreground border-border",
  BLOCKED: "bg-destructive/10 text-destructive border-destructive/20",
  EXHAUSTED: "bg-warning/10 text-warning border-warning/20",
};

interface BadgeProps {
  children: React.ReactNode;
  status?: string;
  className?: string;
}

export function Badge({ children, status, className }: BadgeProps) {
  const colorClass = status ? statusColors[status] || "bg-secondary text-secondary-foreground" : "bg-secondary text-secondary-foreground";
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium",
        colorClass,
        className
      )}
    >
      {children}
    </span>
  );
}
