import { useMemo } from "react";
import { canEncodeQrCode, createQrCodeMatrix, getQrCodeSize } from "@/lib/qrCode";
import { cn } from "@/lib/cn";

interface QrCodeSvgProps {
  value: string;
  className?: string;
  size?: number;
}

export function QrCodeSvg({ value, className, size = 112 }: QrCodeSvgProps) {
  const canEncode = canEncodeQrCode(value);
  const matrix = useMemo(() => (canEncode ? createQrCodeMatrix(value) : undefined), [canEncode, value]);
  const moduleCount = getQrCodeSize();
  const quietZone = 4;
  const viewBoxSize = moduleCount + quietZone * 2;
  const path = matrix
    ?.flatMap((row, y) => row.map((enabled, x) => (enabled ? `M${x + quietZone},${y + quietZone}h1v1h-1z` : "")))
    .filter(Boolean)
    .join("");

  if (!matrix || !path) {
    return (
      <div
        className={cn("grid place-items-center rounded border bg-muted text-center text-[10px] text-muted-foreground", className)}
        style={{ width: size, height: size }}
      >
        链接过长
      </div>
    );
  }

  return (
    <svg
      aria-label="APK 下载二维码"
      className={cn("rounded border bg-white p-1", className)}
      height={size}
      role="img"
      viewBox={`0 0 ${viewBoxSize} ${viewBoxSize}`}
      width={size}
    >
      <path d={`M0,0h${viewBoxSize}v${viewBoxSize}H0z`} fill="#fff" />
      <path d={path} fill="#111827" />
    </svg>
  );
}
