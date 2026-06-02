import { Badge } from "@/components/ui/Badge";

export function MockBadge({ show }: { show?: boolean }) {
  if (!show) return null;
  return <Badge tone="warning">MOCK</Badge>;
}
