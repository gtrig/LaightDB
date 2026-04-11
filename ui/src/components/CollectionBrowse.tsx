import { useParams, Link, useNavigate } from "react-router-dom";
import { useApi } from "../hooks/useApi";
import { listEntries } from "../api";
import ContentTypeBadge from "./ContentTypeBadge";
import { useState } from "react";

export default function CollectionBrowse() {
  const { name: rawName } = useParams<{ name: string }>();
  const name = rawName ? decodeURIComponent(rawName) : "";
  const navigate = useNavigate();
  const [limit, setLimit] = useState(200);

  const fetchList = () => listEntries({ collection: name, limit });
  const { data: rows, loading, error, refetch } = useApi(fetchList, [name, limit]);

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
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={4} style={{ padding: 32, textAlign: "center", color: "var(--outline)" }}>
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
                </tr>
              ))}
            {!loading && (rows?.length ?? 0) === 0 && (
              <tr>
                <td colSpan={4} style={{ textAlign: "center", color: "var(--outline)", padding: 32 }}>
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
