import type { InputHTMLAttributes } from "react";
import { cn } from "@/lib/cn";

export function Input({ className, ...props }: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      className={cn(
        "h-9 w-full rounded-lg border border-input bg-white px-3 text-xs outline-none transition focus:border-black focus:ring-2 focus:ring-black/10",
        className,
      )}
      {...props}
    />
  );
}
