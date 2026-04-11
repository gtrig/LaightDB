import type { CSSProperties } from "react";

const colorMap: Record<string, string> = {
  code: "var(--badge-code)",
  conversation: "var(--badge-conversation)",
  doc: "var(--badge-doc)",
  kv: "var(--badge-kv)",
};

export default function ContentTypeBadge({ type }: { type: string }) {
  const color = colorMap[type] ?? "var(--outline)";
  const style: CSSProperties = {
    display: "inline-block",
    fontFamily: "var(--font-label)",
    fontSize: 11,
    fontWeight: 500,
    padding: "2px 10px",
    borderRadius: "var(--radius-full)",
    background: `color-mix(in srgb, ${color} 18%, transparent)`,
    color,
    letterSpacing: "0.02em",
  };
  return <span style={style}>{type}</span>;
}
