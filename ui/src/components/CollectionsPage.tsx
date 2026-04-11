import { useMemo, type CSSProperties } from "react";
import { Link } from "react-router-dom";
import { useApi } from "../hooks/useApi";
import { listCollections, listEntries } from "../api";

const cardStyle: CSSProperties = {
  background: "var(--surface-container-low)",
  borderRadius: "var(--radius)",
  padding: "20px 24px",
  borderLeft: "2px solid var(--primary)",
  display: "flex",
  flexDirection: "column",
  gap: 12,
};

export default function CollectionsPage() {
  const { data: names, loading: namesLoading } = useApi(listCollections);
  const { data: allRows, loading: rowsLoading } = useApi(() => listEntries({ limit: 10000 }));

  const counts = useMemo(() => {
    const m = new Map<string, number>();
    for (const row of allRows ?? []) {
      const c = row.collection || "(default)";
      m.set(c, (m.get(c) ?? 0) + 1);
    }
    return m;
  }, [allRows]);

  const loading = namesLoading || rowsLoading;

  return (
    <div>
      <h1 style={{ fontFamily: "var(--font-headline)", fontSize: 24, fontWeight: 700, marginBottom: 8 }}>
        Collections
      </h1>
      <p style={{ color: "var(--on-surface-variant)", marginBottom: 24, fontSize: 14 }}>
        Browse entries by namespace. Counts reflect loaded data (up to 10k most recent entries).
      </p>

      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fill, minmax(260px, 1fr))",
          gap: 16,
        }}
      >
        {(names ?? []).map((name) => (
          <div key={name} style={cardStyle}>
            <div style={{ fontFamily: "var(--font-headline)", fontWeight: 700, fontSize: 18 }}>{name}</div>
            <div style={{ fontFamily: "var(--font-label)", fontSize: 13, color: "var(--on-surface-variant)" }}>
              {loading ? "…" : `${counts.get(name) ?? 0} entries`}
            </div>
            <Link
              to={`/collections/${encodeURIComponent(name)}`}
              style={{
                alignSelf: "flex-start",
                marginTop: 4,
                fontSize: 13,
                fontWeight: 600,
                color: "var(--primary)",
              }}
            >
              Open →
            </Link>
          </div>
        ))}
        {!loading && (names?.length ?? 0) === 0 && (
          <div style={{ color: "var(--outline)", fontStyle: "italic" }}>No collections yet.</div>
        )}
      </div>
    </div>
  );
}
