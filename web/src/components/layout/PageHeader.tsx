import type { PropsWithChildren } from "react";

export function PageHeader({ title, children }: PropsWithChildren<{ title: string }>) {
  return (
    <div className="mb-3 flex min-h-9 flex-col items-start justify-between gap-2 sm:flex-row sm:items-center lg:pr-64">
      <h1 className="text-lg font-semibold tracking-normal sm:text-xl">{title}</h1>
      <div className="flex w-full flex-wrap items-center gap-2 sm:w-auto sm:justify-end">{children}</div>
    </div>
  );
}
