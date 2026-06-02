import type { HTMLAttributes } from "react";
import { cn } from "@/lib/cn";

export function Card({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <section className={cn("min-w-0 rounded-xl border border-border bg-card p-3.5 shadow-soft", className)} {...props} />;
}
