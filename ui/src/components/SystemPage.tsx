import { useState, type CSSProperties } from "react";
import { useApi } from "../hooks/useApi";
import { getStats, healthCheck, compactCollection } from "../api";

const cardStyle: CSSProperties = {
  background: "var(--surface-container-low)",
  borderRadius: "var(--radius)",
  padding: "20px 24px",
  marginBottom: 16,
  borderLeft: "2px solid var(--primary)",
};

export default function SystemPage() {
  const { data: stats, loading, error, refetch } = useApi(getStats);
  const { data: health } = useApi(healthCheck);
  const [compacting, setCompacting] = useState(false);
  const [compactMsg, setCompactMsg] = useState<string | null>(null);

  async function runCompact() {
    if (!window.confirm("Run storage compaction? This can take a while on large datasets.")) return;
    setCompacting(true);
    setCompactMsg(null);
    try {
      await compactCollection("_");
      setCompactMsg("Compaction accepted (202).");
      refetch();
    } catch (e) {
      setCompactMsg(e instanceof Error ? e.message : "Compaction failed");
    } finally {
      setCompacting(false);
    }
  }

  return (
    <div style={{ maxWidth: 640 }}>
      <h1 style={{ fontFamily: "var(--font-headline)", fontSize: 24, fontWeight: 700, marginBottom: 8 }}>
        System
      </h1>
      <p style={{ color: "var(--on-surface-variant)", marginBottom: 24, fontSize: 14 }}>
        API status, database stats, and maintenance.
      </p>

      <div style={cardStyle}>
        <div style={{ fontFamily: "var(--font-label)", fontSize: 11, letterSpacing: "0.05em", color: "var(--outline)", marginBottom: 8 }}>
          API HEALTH
        </div>
        <div style={{ fontSize: 16, fontWeight: 600, color: health?.status === "ok" ? "#4ade80" : "var(--error)" }}>
          {health?.status === "ok" ? "OK" : health ? "Unhealthy" : "…"}
        </div>
      </div>

      <div style={cardStyle}>
        <div style={{ fontFamily: "var(--font-label)", fontSize: 11, letterSpacing: "0.05em", color: "var(--outline)", marginBottom: 12 }}>
          STATS
        </div>
        {error && <div style={{ color: "var(--error)" }}>{error}</div>}
        {loading && !stats && <div style={{ color: "var(--outline)" }}>Loading…</div>}
        {stats && (
          <ul style={{ listStyle: "none", padding: 0, margin: 0, fontSize: 14, lineHeight: 2 }}>
            <li>
              <strong>Entries:</strong> {stats.entries?.toLocaleString?.() ?? stats.entries}
            </li>
            <li>
              <strong>Collections:</strong> {stats.collections?.toLocaleString?.() ?? stats.collections}
            </li>
            <li>
              <strong>Vector nodes:</strong> {stats.vector_nodes?.toLocaleString?.() ?? stats.vector_nodes}
            </li>
          </ul>
        )}
        <button
          type="button"
          onClick={() => refetch()}
          style={{
            marginTop: 12,
            background: "var(--surface-container-highest)",
            color: "var(--primary)",
            borderRadius: "var(--radius)",
            padding: "6px 14px",
            fontSize: 13,
          }}
        >
          Refresh stats
        </button>
      </div>

      <div style={cardStyle}>
        <div style={{ fontFamily: "var(--font-label)", fontSize: 11, letterSpacing: "0.05em", color: "var(--outline)", marginBottom: 8 }}>
          STORAGE
        </div>
        <p style={{ fontSize: 13, color: "var(--on-surface-variant)", marginBottom: 12 }}>
          Triggers LSM compaction for the whole engine (collection name in the URL is ignored by the server).
        </p>
        <button
          type="button"
          disabled={compacting}
          onClick={runCompact}
          style={{
            background: "linear-gradient(135deg, var(--primary), var(--primary-container))",
            color: "var(--on-primary-fixed)",
            borderRadius: "var(--radius)",
            padding: "10px 20px",
            fontWeight: 600,
            fontSize: 14,
            opacity: compacting ? 0.6 : 1,
          }}
        >
          {compacting ? "Requesting…" : "Compact storage"}
        </button>
        {compactMsg && (
          <div style={{ marginTop: 10, fontSize: 13, color: "var(--on-surface-variant)" }}>{compactMsg}</div>
        )}
      </div>
    </div>
  );
}
