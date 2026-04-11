import { useApi } from "../hooks/useApi";
import { getStats, listCollections, listEntries } from "../api";
import ContentTypeBadge from "./ContentTypeBadge";
import CollectionBadge from "./CollectionBadge";
import { useNavigate } from "react-router-dom";
import { useState, useEffect, type CSSProperties } from "react";
import { getContext } from "../api";
import type { ContextEntry } from "../types";

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
  const navigate = useNavigate();

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
              </tr>
            ))}
            {loadingRecent && !recentEntries.length && (
              <tr>
                <td colSpan={5} style={{ textAlign: "center", color: "var(--outline)", padding: 32 }}>
                  Loading…
                </td>
              </tr>
            )}
            {!loadingRecent && recentEntries.length === 0 && (
              <tr>
                <td colSpan={5} style={{ textAlign: "center", color: "var(--outline)", padding: 32 }}>
                  No entries yet
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
