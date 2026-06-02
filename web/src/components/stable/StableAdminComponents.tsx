import { FileWarning } from "lucide-react";
import { isApiError } from "@/api/errors";
import { Badge } from "@/components/ui/Badge";
import { Card } from "@/components/ui/Card";

export function ApiErrorState({ error, title = "加载失败" }: { error: unknown; title?: string }) {
  const message = error instanceof Error ? error.message : "请求失败";
  const requestId = isApiError(error) ? error.requestId : undefined;
  const code = isApiError(error) ? error.code : undefined;
  return (
    <Card className="border-red-200 bg-red-50 text-red-800">
      <div className="flex items-start gap-2">
        <FileWarning className="mt-0.5 h-4 w-4 shrink-0" />
        <div className="min-w-0">
          <div className="font-medium">{title}</div>
          <div className="mt-1 text-xs leading-5">{message}</div>
          {code || requestId ? (
            <div className="mt-2 flex flex-wrap gap-2 text-[11px]">
              {code ? <Badge tone="danger">{code}</Badge> : null}
              {requestId ? <span className="font-mono">{requestId}</span> : null}
            </div>
          ) : null}
        </div>
      </div>
    </Card>
  );
}
