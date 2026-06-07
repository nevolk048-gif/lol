"use client";

import { useRouter } from "next/navigation";
import { Command } from "cmdk";
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
} from "lucide-react";
import { cn } from "@/lib/utils";

const pages = [
  { name: "Dashboard", href: "/", icon: LayoutDashboard },
  { name: "Transactions", href: "/transactions", icon: ArrowLeftRight },
  { name: "Providers", href: "/providers", icon: Building2 },
  { name: "Casinos", href: "/casinos", icon: CreditCard },
  { name: "Requisites", href: "/requisites", icon: CreditCard },
  { name: "Routing", href: "/routing", icon: Route },
  { name: "Finance", href: "/finance", icon: DollarSign },
  { name: "Monitoring", href: "/monitoring", icon: Activity },
  { name: "Integration Logs", href: "/integration-logs", icon: FileText },
  { name: "Admin", href: "/admin", icon: Users },
  { name: "Sandbox", href: "/sandbox", icon: FlaskConical },
];

interface CommandPaletteProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CommandPalette({ open, onOpenChange }: CommandPaletteProps) {
  const router = useRouter();

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50">
      <div
        className="fixed inset-0 bg-background/80 backdrop-blur-sm"
        onClick={() => onOpenChange(false)}
      />
      <div className="fixed left-1/2 top-[20%] z-50 w-full max-w-lg -translate-x-1/2">
        <Command
          className="rounded-xl border border-border bg-popover shadow-2xl overflow-hidden"
          loop
        >
          <div className="flex items-center border-b border-border px-4">
            <Command.Input
              placeholder="Type a command or search..."
              className="flex h-12 w-full bg-transparent text-sm outline-none placeholder:text-muted-foreground"
            />
          </div>
          <Command.List className="max-h-80 overflow-y-auto p-2">
            <Command.Empty className="py-6 text-center text-sm text-muted-foreground">
              No results found.
            </Command.Empty>
            <Command.Group heading="Navigation" className="text-xs text-muted-foreground px-2 py-1.5">
              {pages.map((page) => (
                <Command.Item
                  key={page.href}
                  value={page.name}
                  onSelect={() => {
                    router.push(page.href);
                    onOpenChange(false);
                  }}
                  className={cn(
                    "flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm cursor-pointer",
                    "aria-selected:bg-accent aria-selected:text-accent-foreground"
                  )}
                >
                  <page.icon className="h-4 w-4 text-muted-foreground" />
                  {page.name}
                </Command.Item>
              ))}
            </Command.Group>
          </Command.List>
        </Command>
      </div>
    </div>
  );
}
