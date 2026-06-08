"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { motion } from "framer-motion";
import {
  LayoutDashboard,
  ArrowLeftRight,
  Building2,
  CreditCard,
  Route,
  DollarSign,
  Activity,
  Users,
  FlaskConical,
  FileText,
  Zap,
  ChevronLeft,
  ChevronRight,
  AlertTriangle,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useState } from "react";
import { useI18n } from "@/hooks/use-i18n";

const navigationKeys = [
  { key: "dashboard", href: "/", icon: LayoutDashboard },
  { key: "transactions", href: "/transactions", icon: ArrowLeftRight },
  { key: "providers", href: "/providers", icon: Building2 },
  { key: "casinos", href: "/casinos", icon: CreditCard },
  { key: "requisites", href: "/requisites", icon: CreditCard },
  { key: "routing", href: "/routing", icon: Route },
  { key: "disputes", href: "/disputes", icon: AlertTriangle },
  { key: "finance", href: "/finance", icon: DollarSign },
  { key: "monitoring", href: "/monitoring", icon: Activity },
  { key: "integrationLogs", href: "/integration-logs", icon: FileText },
  { key: "admin", href: "/admin", icon: Users },
  { key: "sandbox", href: "/sandbox", icon: FlaskConical },
];

export function Sidebar() {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState(false);
  const { t } = useI18n();

  return (
    <motion.aside
      initial={false}
      animate={{ width: collapsed ? 72 : 260 }}
      transition={{ duration: 0.2 }}
      className="fixed left-0 top-0 z-40 flex h-screen flex-col border-r border-border bg-sidebar"
    >
      <div className="flex h-16 items-center gap-3 border-b border-border px-4">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary">
          <Zap className="h-5 w-5 text-primary-foreground" />
        </div>
        {!collapsed && (
          <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
            <p className="text-sm font-bold tracking-tight">PaymentsGate</p>
            <p className="text-[10px] text-muted-foreground">{t("enterpriseAggregator")}</p>
          </motion.div>
        )}
      </div>

      <nav className="flex-1 space-y-1 overflow-y-auto p-3">
        {navigationKeys.map((item) => {
          const isActive =
            item.href === "/"
              ? pathname === "/"
              : pathname.startsWith(item.href);
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors",
                isActive
                  ? "bg-primary/10 text-primary"
                  : "text-sidebar-foreground hover:bg-accent hover:text-accent-foreground"
              )}
            >
              <item.icon className="h-4 w-4 shrink-0" />
              {!collapsed && <span>{t(item.key)}</span>}
            </Link>
          );
        })}
      </nav>

      <button
        onClick={() => setCollapsed(!collapsed)}
        className="flex h-12 items-center justify-center border-t border-border text-muted-foreground hover:text-foreground transition-colors"
      >
        {collapsed ? <ChevronRight className="h-4 w-4" /> : <ChevronLeft className="h-4 w-4" />}
      </button>
    </motion.aside>
  );
}
