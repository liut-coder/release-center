import type { TableHTMLAttributes } from "react";
import { cn } from "@/lib/cn";

export function Table({ className, ...props }: TableHTMLAttributes<HTMLTableElement>) {
  return (
    <div className="-mx-1 overflow-x-auto px-1">
      <table className={cn("w-full min-w-[680px] table-fixed border-separate border-spacing-0 text-xs", className)} {...props} />
    </div>
  );
}

export function Th({ className, ...props }: React.ThHTMLAttributes<HTMLTableCellElement>) {
  return <th className={cn("border-b px-2.5 py-2 text-left text-[11px] font-medium text-muted-foreground", className)} {...props} />;
}

export function Td({ className, ...props }: React.TdHTMLAttributes<HTMLTableCellElement>) {
  return <td className={cn("truncate border-b px-2.5 py-2 text-foreground", className)} {...props} />;
}
