import { useParams, Link, useNavigate } from "react-router-dom";
import { useApi } from "../hooks/useApi";
import { useAuth } from "../hooks/useAuth";
import { listEntries, deleteContext, deleteCollection } from "../api";
import ContentTypeBadge from "./ContentTypeBadge";
import { useState } from "react";

export default function CollectionBrowse() {
  const { name: rawName } = useParams<{ name: string }>();
  const name = rawName ? decodeURIComponent(rawName) : "";
  const navigate = useNavigate();
  const [limit, setLimit] = useState(200);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [deletingAll, setDeletingAll] = useState(false);
  const { user, authRequired } = useAuth();
  const isAdmin = !authRequired || user?.role === "admin";

  const fetchList = () => listEntries({ collection: name, limit });
  const { data: rows, loading, error, refetch } = useApi(fetchList, [name, limit]);

  async function handleDeleteEntry(id: string, e: React.MouseEvent) {
    e.stopPropagation();
    if (!window.confirm("Delete this entry? This cannot be undone.")) return;
    setDeletingId(id);
    try {
      await deleteContext(id);
      refetch();
    } catch {
      /* error silently */
    } finally {
      setDeletingId(null);
    }
  }

  async function handleDeleteCollection() {
    if (!name) return;
    const count = rows?.length ?? 0;
    if (!window.confirm(`Delete the entire "${name}" collection (${count} entries)? This cannot be undone.`)) return;
    setDeletingAll(true);
    try {
      await deleteCollection(name);
      navigate("/collections");
    } catch {
      setDeletingAll(false);
    }
  }

  function timeAgo(dateStr: string): string {
    const diff = Date.now() - new Date(dateStr).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return "just now";
    if (mins < 60) return `${mins}m ago`;
    const hours = Math.floor(mins / 60);
    if (hours < 24) return `${hours}h ago`;
    return `${Math.floor(hours / 24)}d ago`;
  }

  if (!name) return null;

  return (
    <div>
      <div style={{ marginBottom: 20 }}>
        <Link to="/collections" style={{ fontSize: 13, color: "var(--primary)" }}>
          ← All collections
        </Link>
      </div>
      <h1 style={{ fontFamily: "var(--font-headline)", fontSize: 24, fontWeight: 700, marginBottom: 16 }}>
        {name}
      </h1>

      <div style={{ display: "flex", alignItems: "center", gap: 16, marginBottom: 20, flexWrap: "wrap" }}>
        <label style={{ fontSize: 13, color: "var(--on-surface-variant)" }}>
          Limit{" "}
          <select
            value={limit}
            onChange={(e) => setLimit(Number(e.target.value))}
            style={{ marginLeft: 8 }}
          >
            {[50, 100, 200, 500, 1000].map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
          </select>
        </label>
        <button
          type="button"
          onClick={() => refetch()}
          style={{
            background: "var(--surface-container-highest)",
            color: "var(--primary)",
            borderRadius: "var(--radius)",
            padding: "6px 14px",
            fontSize: 13,
          }}
        >
          Refresh
        </button>
        {isAdmin && (
          <button
            type="button"
            onClick={handleDeleteCollection}
            disabled={deletingAll}
            style={{
              background: "rgba(255, 180, 171, 0.12)",
              color: "var(--error)",
              borderRadius: "var(--radius)",
              padding: "6px 14px",
              fontSize: 13,
              fontWeight: 500,
              display: "flex",
              alignItems: "center",
              gap: 6,
              opacity: deletingAll ? 0.5 : 1,
              marginLeft: "auto",
            }}
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M3 6h18M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2m3 0v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6h14" />
            </svg>
            {deletingAll ? "Deleting…" : "Delete Collection"}
          </button>
        )}
      </div>

      {error && <div style={{ color: "var(--error)", marginBottom: 12 }}>{error}</div>}

      <div style={{ background: "var(--surface-container-low)", borderRadius: "var(--radius)", overflow: "hidden" }}>
        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>Type</th>
              <th>Tokens</th>
              <th>Updated</th>
              {isAdmin && <th style={{ width: 60 }}></th>}
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={isAdmin ? 5 : 4} style={{ padding: 32, textAlign: "center", color: "var(--outline)" }}>
                  Loading…
                </td>
              </tr>
            )}
            {!loading &&
              rows?.map((row) => (
                <tr
                  key={row.id}
                  style={{ cursor: "pointer" }}
                  onClick={() => navigate(`/contexts/${row.id}`)}
                >
                  <td>
                    <span style={{ fontFamily: "var(--font-label)", fontSize: 12, color: "var(--primary)" }}>
                      {row.id.substring(0, 10)}…
                    </span>
                  </td>
                  <td>
                    <ContentTypeBadge type={row.content_type} />
                  </td>
                  <td style={{ fontFamily: "var(--font-label)", fontSize: 12 }}>{row.token_count}</td>
                  <td style={{ fontSize: 12, color: "var(--on-surface-variant)" }}>{timeAgo(row.updated_at)}</td>
                  {isAdmin && (
                    <td>
                      <button
                        type="button"
                        onClick={(e) => handleDeleteEntry(row.id, e)}
                        disabled={deletingId === row.id}
                        style={{
                          background: "transparent",
                          color: "var(--error)",
                          padding: "4px 8px",
                          fontSize: 12,
                          borderRadius: "var(--radius)",
                          opacity: deletingId === row.id ? 0.5 : 0.7,
                          transition: "opacity 0.15s",
                          cursor: deletingId === row.id ? "default" : "pointer",
                        }}
                        title="Delete entry"
                      >
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                          <path d="M3 6h18M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2m3 0v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6h14" />
                        </svg>
                      </button>
                    </td>
                  )}
                </tr>
              ))}
            {!loading && (rows?.length ?? 0) === 0 && (
              <tr>
                <td colSpan={isAdmin ? 5 : 4} style={{ textAlign: "center", color: "var(--outline)", padding: 32 }}>
                  No entries in this collection.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
