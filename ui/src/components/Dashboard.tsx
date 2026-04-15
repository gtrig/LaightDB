import { useApi } from "../hooks/useApi";
import { useAuth } from "../hooks/useAuth";
import { getStats, listCollections, listEntries, deleteContext } from "../api";
import ContentTypeBadge from "./ContentTypeBadge";
import CollectionBadge from "./CollectionBadge";
import { useNavigate } from "react-router-dom";
import { useState, useEffect, type CSSProperties } from "react";
import { getContext, listAuditCalls } from "../api";
import type { CallLogEntry, ContextEntry } from "../types";
import { Link } from "react-router-dom";

const cardStyle: CSSProperties = {
  flex: 1,
  background: "var(--surface-container-low)",
  borderRadius: "var(--radius)",
  padding: "20px 24px",
  borderLeft: "2px solid var(--primary)",
};

const cardLabel: CSSProperties = {
  fontFamily: "var(--font-label)",
  fontSize: 11,
  fontWeight: 500,
  letterSpacing: "0.05em",
  textTransform: "uppercase",
  color: "var(--on-surface-variant)",
  marginBottom: 6,
};

const cardValue: CSSProperties = {
  fontFamily: "var(--font-headline)",
  fontSize: 28,
  fontWeight: 800,
  color: "var(--on-surface)",
};

export default function Dashboard() {
  const { data: stats, loading: statsLoading } = useApi(getStats);
  const { data: collections } = useApi(listCollections);
  const [recentEntries, setRecentEntries] = useState<ContextEntry[]>([]);
  const [loadingRecent, setLoadingRecent] = useState(true);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const navigate = useNavigate();
  const { user, authRequired } = useAuth();
  const isAdmin = !authRequired || user?.role === "admin";
  const showAuditPreview = authRequired && user?.role === "admin";
  const { data: recentCalls, loading: auditLoading, error: auditError } = useApi(
    () => (showAuditPreview ? listAuditCalls(10) : Promise.resolve([] as CallLogEntry[])),
    [showAuditPreview]
  );

  useEffect(() => {
    setLoadingRecent(true);
    listEntries({ limit: 15 })
      .then((rows) =>
        Promise.all(rows.map((r) => getContext(r.id, "metadata").catch(() => null)))
      )
      .then((entries) => setRecentEntries(entries.filter((e): e is ContextEntry => e !== null)))
      .catch(() => setRecentEntries([]))
      .finally(() => setLoadingRecent(false));
  }, []);

  async function handleDelete(id: string, e: React.MouseEvent) {
    e.stopPropagation();
    if (!window.confirm("Delete this entry? This cannot be undone.")) return;
    setDeletingId(id);
    try {
      await deleteContext(id);
      setRecentEntries((prev) => prev.filter((entry) => entry.id !== id));
    } catch {
      /* error silently */
    } finally {
      setDeletingId(null);
    }
  }

  function timeAgo(dateStr: string): string {
    const diff = Date.now() - new Date(dateStr).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return "just now";
    if (mins < 60) return `${mins}m ago`;
    const hours = Math.floor(mins / 60);
    if (hours < 24) return `${hours}h ago`;
    const days = Math.floor(hours / 24);
    return `${days}d ago`;
  }

  return (
    <div>
      <h1
        style={{
          fontFamily: "var(--font-headline)",
          fontSize: 24,
          fontWeight: 700,
          marginBottom: 24,
        }}
      >
        Dashboard
      </h1>

      <div style={{ display: "flex", gap: 16, marginBottom: 32 }}>
        <div style={cardStyle}>
          <div style={cardLabel}>Entries</div>
          <div style={cardValue}>{statsLoading ? "..." : stats?.entries?.toLocaleString() ?? "0"}</div>
        </div>
        <div style={cardStyle}>
          <div style={cardLabel}>Collections</div>
          <div style={cardValue}>{statsLoading ? "..." : (collections?.length ?? stats?.collections ?? 0)}</div>
        </div>
        <div style={cardStyle}>
          <div style={cardLabel}>Vector Nodes</div>
          <div style={cardValue}>{statsLoading ? "..." : stats?.vector_nodes?.toLocaleString() ?? "0"}</div>
        </div>
      </div>

      <h2
        style={{
          fontFamily: "var(--font-headline)",
          fontSize: 16,
          fontWeight: 700,
          marginBottom: 16,
        }}
      >
        Recent Entries
      </h2>

      <div style={{ background: "var(--surface-container-low)", borderRadius: "var(--radius)", overflow: "hidden" }}>
        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>Collection</th>
              <th>Type</th>
              <th>Tokens</th>
              <th>Created</th>
              {isAdmin && <th style={{ width: 60 }}></th>}
            </tr>
          </thead>
          <tbody>
            {!loadingRecent && recentEntries.map((entry) => (
              <tr
                key={entry.id}
                style={{ cursor: "pointer" }}
                onClick={() => navigate(`/contexts/${entry.id}`)}
              >
                <td>
                  <span style={{ fontFamily: "var(--font-label)", fontSize: 12, color: "var(--primary)" }}>
                    {entry.id.substring(0, 8)}...
                  </span>
                </td>
                <td><CollectionBadge name={entry.collection} /></td>
                <td><ContentTypeBadge type={entry.content_type} /></td>
                <td style={{ fontFamily: "var(--font-label)", fontSize: 12 }}>{entry.token_count}</td>
                <td style={{ fontSize: 12, color: "var(--on-surface-variant)" }}>{timeAgo(entry.created_at)}</td>
                {isAdmin && (
                  <td>
                    <button
                      type="button"
                      onClick={(e) => handleDelete(entry.id, e)}
                      disabled={deletingId === entry.id}
                      style={{
                        background: "transparent",
                        color: "var(--error)",
                        padding: "4px 8px",
                        fontSize: 12,
                        borderRadius: "var(--radius)",
                        opacity: deletingId === entry.id ? 0.5 : 0.7,
                        transition: "opacity 0.15s",
                        cursor: deletingId === entry.id ? "default" : "pointer",
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
            {loadingRecent && !recentEntries.length && (
              <tr>
                <td colSpan={isAdmin ? 6 : 5} style={{ textAlign: "center", color: "var(--outline)", padding: 32 }}>
                  Loading…
                </td>
              </tr>
            )}
            {!loadingRecent && recentEntries.length === 0 && (
              <tr>
                <td colSpan={isAdmin ? 6 : 5} style={{ textAlign: "center", color: "var(--outline)", padding: 32 }}>
                  No entries yet
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {showAuditPreview && (
        <>
          <h2
            style={{
              fontFamily: "var(--font-headline)",
              fontSize: 16,
              fontWeight: 700,
              marginTop: 40,
              marginBottom: 8,
              display: "flex",
              alignItems: "center",
              justifyContent: "space-between",
              gap: 16,
            }}
          >
            Recent MCP activity
            <Link
              to="/audit"
              style={{
                fontSize: 13,
                fontWeight: 600,
                color: "var(--primary)",
                textDecoration: "none",
              }}
            >
              View all
            </Link>
          </h2>
          <p style={{ color: "var(--on-surface-variant)", marginBottom: 16, fontSize: 13 }}>
            MCP tool calls (newest first).
          </p>
          <div style={{ background: "var(--surface-container-low)", borderRadius: "var(--radius)", overflow: "hidden" }}>
            {auditError && (
              <div style={{ padding: 16, color: "var(--error)", fontSize: 13 }}>{auditError}</div>
            )}
            {!auditError && (
              <table>
                <thead>
                  <tr>
                    <th>Time</th>
                    <th>Who</th>
                    <th>Tool</th>
                    <th style={{ textAlign: "right" }}>ms</th>
                  </tr>
                </thead>
                <tbody>
                  {auditLoading && (!recentCalls || recentCalls.length === 0) && (
                    <tr>
                      <td colSpan={4} style={{ textAlign: "center", color: "var(--outline)", padding: 24 }}>
                        Loading…
                      </td>
                    </tr>
                  )}
                  {!auditLoading && recentCalls && recentCalls.length === 0 && (
                    <tr>
                      <td colSpan={4} style={{ textAlign: "center", color: "var(--outline)", padding: 24 }}>
                        No calls recorded yet
                      </td>
                    </tr>
                  )}
                  {recentCalls?.map((c) => (
                    <tr key={c.id}>
                      <td style={{ fontSize: 12, color: "var(--on-surface-variant)" }}>
                        {new Date(c.ts).toLocaleString()}
                      </td>
                      <td style={{ fontSize: 12 }}>{c.username ?? "—"}</td>
                      <td style={{ fontSize: 12 }}>{c.tool ?? "—"}</td>
                      <td style={{ textAlign: "right", fontSize: 12 }}>{c.duration_ms}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </>
      )}
    </div>
  );
}
