import type { HTMLAttributes } from "react";
import { cn } from "@/lib/cn";

type Tone = "default" | "success" | "warning" | "danger";

const tones: Record<Tone, string> = {
  default: "border-border bg-muted text-muted-foreground",
  success: "border-black/10 bg-black text-white",
  warning: "border-zinc-300 bg-zinc-100 text-zinc-700",
  danger: "border-red-200 bg-red-50 text-red-700",
};

export function Badge({ className, tone = "default", ...props }: HTMLAttributes<HTMLSpanElement> & { tone?: Tone }) {
  return (
    <span
      className={cn("inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium", tones[tone], className)}
      {...props}
    />
  );
}
