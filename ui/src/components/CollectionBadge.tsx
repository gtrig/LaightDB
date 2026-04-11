import type { CSSProperties } from "react";

export default function CollectionBadge({ name }: { name: string }) {
  const style: CSSProperties = {
    display: "inline-block",
    fontFamily: "var(--font-label)",
    fontSize: 11,
    fontWeight: 500,
    padding: "2px 10px",
    borderRadius: "var(--radius-full)",
    background: "var(--secondary-container)",
    color: "var(--on-secondary-container)",
    letterSpacing: "0.02em",
  };
  return <span style={style}>{name}</span>;
}
