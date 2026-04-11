import { useState, useCallback, type CSSProperties } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { getContext, deleteContext } from "../api";
import { useApi } from "../hooks/useApi";
import ContentTypeBadge from "./ContentTypeBadge";
import CollectionBadge from "./CollectionBadge";
import type { DetailLevel } from "../types";

const tabs: { label: string; detail: DetailLevel }[] = [
  { label: "Metadata", detail: "metadata" },
  { label: "Summary", detail: "summary" },
  { label: "Full Content", detail: "full" },
];

const tabBaseStyle: CSSProperties = {
  padding: "10px 20px",
  fontSize: 13,
  fontWeight: 500,
  background: "transparent",
  color: "var(--on-surface-variant)",
  borderRadius: "var(--radius) var(--radius) 0 0",
  transition: "all 0.15s",
  cursor: "pointer",
};

const tabActiveStyle: CSSProperties = {
  ...tabBaseStyle,
  background: "var(--surface-container-high)",
  color: "var(--primary)",
};

const kvRowStyle = (even: boolean): CSSProperties => ({
  display: "flex",
  padding: "10px 16px",
  background: even ? "var(--surface-container-low)" : "transparent",
  borderRadius: "var(--radius)",
});

export default function ContextDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState(0);
  const [deleting, setDeleting] = useState(false);

  const detailLevel: DetailLevel = activeTab === 3 ? "full" : tabs[activeTab].detail;
  const fetchEntry = useCallback(() => getContext(id!, detailLevel), [id, detailLevel]);
  const { data: entry, loading, error } = useApi(fetchEntry, [id, detailLevel]);

  async function copyId() {
    if (!id) return;
    try {
      await navigator.clipboard.writeText(id);
    } catch {
      /* ignore */
    }
  }

  async function handleDelete() {
    if (!id) return;
    if (!window.confirm("Delete this context entry? This cannot be undone.")) return;
    setDeleting(true);
    try {
      await deleteContext(id);
      navigate("/");
    } catch {
      setDeleting(false);
    }
  }

  if (loading) {
    return <div style={{ color: "var(--outline)", padding: 48, textAlign: "center" }}>Loading...</div>;
  }

  if (error || !entry) {
    return <div style={{ color: "var(--error)", padding: 48, textAlign: "center" }}>{error ?? "Not found"}</div>;
  }

  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 24, flexWrap: "wrap" }}>
        <span style={{ fontFamily: "var(--font-label)", fontSize: 14, color: "var(--on-surface)", wordBreak: "break-all" }}>
          {entry.id}
        </span>
        <CollectionBadge name={entry.collection} />
        <ContentTypeBadge type={entry.content_type} />
        <div style={{ flex: 1 }} />
        <button
          type="button"
          onClick={copyId}
          style={{
            background: "var(--surface-container-highest)",
            color: "var(--on-surface-variant)",
            borderRadius: "var(--radius)",
            padding: "8px 16px",
            fontSize: 13,
            fontWeight: 500,
          }}
        >
          Copy ID
        </button>
        <button
          onClick={handleDelete}
          disabled={deleting}
          style={{
            background: "rgba(255, 180, 171, 0.12)",
            color: "var(--error)",
            borderRadius: "var(--radius)",
            padding: "8px 16px",
            fontSize: 13,
            fontWeight: 500,
            display: "flex",
            alignItems: "center",
            gap: 6,
            opacity: deleting ? 0.5 : 1,
          }}
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M3 6h18M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2m3 0v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6h14" />
          </svg>
          {deleting ? "Deleting..." : "Delete"}
        </button>
      </div>

      <div style={{ display: "flex", gap: 2, marginBottom: 0 }}>
        {tabs.map((tab, i) => (
          <button
            key={tab.label}
            onClick={() => setActiveTab(i)}
            style={i === activeTab ? tabActiveStyle : tabBaseStyle}
          >
            {tab.label}
          </button>
        ))}
        {entry.chunks && entry.chunks.length > 0 && (
          <button
            onClick={() => setActiveTab(3)}
            style={activeTab === 3 ? tabActiveStyle : tabBaseStyle}
          >
            Chunks ({entry.chunks.length})
          </button>
        )}
      </div>

      <div style={{ background: "var(--surface-container-high)", borderRadius: "0 var(--radius) var(--radius) var(--radius)", padding: 24, minHeight: 200 }}>
        {activeTab === 0 && (
          <div>
            <div style={{ marginBottom: 20 }}>
              {Object.entries(entry.metadata ?? {}).map(([key, value], i) => (
                <div key={key} style={kvRowStyle(i % 2 === 0)}>
                  <span style={{ fontFamily: "var(--font-label)", fontSize: 12, color: "var(--on-surface-variant)", width: 160, flexShrink: 0, textTransform: "uppercase", letterSpacing: "0.04em" }}>
                    {key}
                  </span>
                  <span style={{ fontSize: 13, color: "var(--on-surface)" }}>{value}</span>
                </div>
              ))}
              {(!entry.metadata || Object.keys(entry.metadata).length === 0) && (
                <div style={{ color: "var(--outline)", fontStyle: "italic", fontSize: 13 }}>No metadata</div>
              )}
            </div>
            <div style={{ fontFamily: "var(--font-label)", fontSize: 11, letterSpacing: "0.05em", textTransform: "uppercase", color: "var(--outline)", marginBottom: 10 }}>
              System Info
            </div>
            <div style={kvRowStyle(true)}>
              <span style={{ fontFamily: "var(--font-label)", fontSize: 12, color: "var(--on-surface-variant)", width: 160 }}>CREATED AT</span>
              <span style={{ fontSize: 13 }}>{new Date(entry.created_at).toLocaleString()}</span>
            </div>
            <div style={kvRowStyle(false)}>
              <span style={{ fontFamily: "var(--font-label)", fontSize: 12, color: "var(--on-surface-variant)", width: 160 }}>UPDATED AT</span>
              <span style={{ fontSize: 13 }}>{new Date(entry.updated_at).toLocaleString()}</span>
            </div>
            <div style={kvRowStyle(true)}>
              <span style={{ fontFamily: "var(--font-label)", fontSize: 12, color: "var(--on-surface-variant)", width: 160 }}>TOKEN COUNT</span>
              <span style={{ fontFamily: "var(--font-label)", fontSize: 13 }}>{entry.token_count}</span>
            </div>
          </div>
        )}

        {activeTab === 1 && (
          <div style={{ fontSize: 14, lineHeight: 1.7, color: "var(--on-surface)" }}>
            {entry.summary || <span style={{ color: "var(--outline)", fontStyle: "italic" }}>No summary available</span>}
          </div>
        )}

        {activeTab === 2 && (
          <pre
            style={{
              fontFamily: "var(--font-label)",
              fontSize: 13,
              lineHeight: 1.6,
              whiteSpace: "pre-wrap",
              wordBreak: "break-word",
              color: "var(--on-surface)",
              background: "var(--surface-container-lowest)",
              padding: 20,
              borderRadius: "var(--radius)",
              maxHeight: 600,
              overflow: "auto",
            }}
          >
            {entry.content || <span style={{ color: "var(--outline)", fontStyle: "italic" }}>No content available at this detail level</span>}
          </pre>
        )}

        {activeTab === 3 && entry.chunks && (
          <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
            {entry.chunks.map((chunk) => (
              <div key={chunk.Index} style={{ background: "var(--surface-container-lowest)", borderRadius: "var(--radius)", padding: 16 }}>
                <div style={{ fontFamily: "var(--font-label)", fontSize: 11, color: "var(--outline)", marginBottom: 8 }}>
                  Chunk {chunk.Index}
                </div>
                <pre style={{ fontFamily: "var(--font-label)", fontSize: 12, lineHeight: 1.5, whiteSpace: "pre-wrap", wordBreak: "break-word", color: "var(--on-surface)" }}>
                  {chunk.Text}
                </pre>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
