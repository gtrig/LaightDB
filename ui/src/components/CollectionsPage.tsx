import { useMemo, useState, type CSSProperties } from "react";
import { Link } from "react-router-dom";
import { useApi } from "../hooks/useApi";
import { useAuth } from "../hooks/useAuth";
import { listCollections, listEntries, deleteCollection } from "../api";

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
  const { data: names, loading: namesLoading, refetch } = useApi(listCollections);
  const { data: allRows, loading: rowsLoading, refetch: refetchRows } = useApi(() => listEntries({ limit: 10000 }));
  const [deletingName, setDeletingName] = useState<string | null>(null);
  const { user, authRequired } = useAuth();
  const isAdmin = !authRequired || user?.role === "admin";

  const counts = useMemo(() => {
    const m = new Map<string, number>();
    for (const row of allRows ?? []) {
      const c = row.collection || "(default)";
      m.set(c, (m.get(c) ?? 0) + 1);
    }
    return m;
  }, [allRows]);

  const loading = namesLoading || rowsLoading;

  async function handleDeleteCollection(name: string) {
    const count = counts.get(name) ?? 0;
    if (!window.confirm(`Delete the entire "${name}" collection (${count} entries)? This cannot be undone.`)) return;
    setDeletingName(name);
    try {
      await deleteCollection(name);
      refetch();
      refetchRows();
    } catch {
      /* error silently */
    } finally {
      setDeletingName(null);
    }
  }

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
        {(names ?? []).map((collName) => (
          <div key={collName} style={cardStyle}>
            <div style={{ fontFamily: "var(--font-headline)", fontWeight: 700, fontSize: 18 }}>{collName}</div>
            <div style={{ fontFamily: "var(--font-label)", fontSize: 13, color: "var(--on-surface-variant)" }}>
              {loading ? "…" : `${counts.get(collName) ?? 0} entries`}
            </div>
            <div style={{ display: "flex", alignItems: "center", gap: 12, marginTop: 4 }}>
              <Link
                to={`/collections/${encodeURIComponent(collName)}`}
                style={{
                  fontSize: 13,
                  fontWeight: 600,
                  color: "var(--primary)",
                }}
              >
                Open →
              </Link>
              {isAdmin && (
                <button
                  type="button"
                  onClick={() => handleDeleteCollection(collName)}
                  disabled={deletingName === collName}
                  style={{
                    background: "transparent",
                    color: "var(--error)",
                    padding: "4px 8px",
                    fontSize: 12,
                    borderRadius: "var(--radius)",
                    opacity: deletingName === collName ? 0.5 : 0.7,
                    transition: "opacity 0.15s",
                    cursor: deletingName === collName ? "default" : "pointer",
                    marginLeft: "auto",
                    display: "flex",
                    alignItems: "center",
                    gap: 4,
                  }}
                  title={`Delete ${collName} collection`}
                >
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M3 6h18M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2m3 0v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6h14" />
                  </svg>
                  {deletingName === collName ? "…" : "Delete"}
                </button>
              )}
            </div>
          </div>
        ))}
        {!loading && (names?.length ?? 0) === 0 && (
          <div style={{ color: "var(--outline)", fontStyle: "italic" }}>No collections yet.</div>
        )}
      </div>
    </div>
  );
}
