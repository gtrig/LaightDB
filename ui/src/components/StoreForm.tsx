import { useState, type FormEvent, type CSSProperties } from "react";
import { storeContext, listCollections } from "../api";
import { useApi } from "../hooks/useApi";
import { useNavigate } from "react-router-dom";
import type { ContentType } from "../types";

const fieldStyle: CSSProperties = { marginBottom: 20 };

const labelStyle: CSSProperties = {
  display: "block",
  fontFamily: "var(--font-label)",
  fontSize: 11,
  fontWeight: 500,
  letterSpacing: "0.05em",
  textTransform: "uppercase",
  color: "var(--on-surface-variant)",
  marginBottom: 8,
};

const contentTypes: ContentType[] = ["code", "conversation", "doc", "kv"];

export default function StoreForm() {
  const [collection, setCollection] = useState("");
  const [content, setContent] = useState("");
  const [contentType, setContentType] = useState<ContentType>("doc");
  const [metaPairs, setMetaPairs] = useState<{ key: string; value: string }[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { data: collections } = useApi(listCollections);
  const navigate = useNavigate();

  function addMeta() {
    setMetaPairs([...metaPairs, { key: "", value: "" }]);
  }

  function removeMeta(i: number) {
    setMetaPairs(metaPairs.filter((_, idx) => idx !== i));
  }

  function updateMeta(i: number, field: "key" | "value", val: string) {
    const updated = [...metaPairs];
    updated[i] = { ...updated[i], [field]: val };
    setMetaPairs(updated);
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!collection.trim() || !content.trim()) return;

    setSubmitting(true);
    setError(null);
    try {
      const metadata: Record<string, string> = {};
      for (const p of metaPairs) {
        if (p.key.trim() && p.value.trim()) metadata[p.key] = p.value;
      }
      const result = await storeContext({
        collection: collection.trim(),
        content,
        content_type: contentType,
        metadata: Object.keys(metadata).length > 0 ? metadata : undefined,
      });
      navigate(`/contexts/${result.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to store context");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div style={{ maxWidth: 720 }}>
      <h1 style={{ fontFamily: "var(--font-headline)", fontSize: 24, fontWeight: 700, marginBottom: 24 }}>
        Store Context
      </h1>

      <div style={{ background: "var(--surface-container-low)", borderRadius: "var(--radius)", padding: 32 }}>
        <form onSubmit={handleSubmit}>
          <div style={fieldStyle}>
            <label style={labelStyle}>Collection</label>
            <input
              list="collections-list"
              value={collection}
              onChange={(e) => setCollection(e.target.value)}
              placeholder="Select or type a new collection..."
              style={{ width: "100%" }}
            />
            <datalist id="collections-list">
              {collections?.map((c) => <option key={c} value={c} />)}
            </datalist>
          </div>

          <div style={fieldStyle}>
            <label style={labelStyle}>Content</label>
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              rows={10}
              placeholder="Paste content here..."
              style={{
                width: "100%",
                fontFamily: "var(--font-label)",
                fontSize: 13,
                lineHeight: 1.6,
                resize: "vertical",
              }}
            />
          </div>

          <div style={fieldStyle}>
            <label style={labelStyle}>Content Type</label>
            <select
              value={contentType}
              onChange={(e) => setContentType(e.target.value as ContentType)}
              style={{ width: 200 }}
            >
              {contentTypes.map((t) => (
                <option key={t} value={t}>{t}</option>
              ))}
            </select>
          </div>

          <div style={fieldStyle}>
            <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 10 }}>
              <label style={{ ...labelStyle, margin: 0 }}>Metadata</label>
              <button
                type="button"
                onClick={addMeta}
                style={{
                  background: "var(--surface-container-highest)",
                  color: "var(--primary)",
                  borderRadius: "var(--radius)",
                  padding: "4px 12px",
                  fontSize: 12,
                  fontWeight: 500,
                }}
              >
                + Add
              </button>
            </div>
            {metaPairs.map((p, i) => (
              <div key={i} style={{ display: "flex", gap: 8, marginBottom: 8 }}>
                <input
                  placeholder="Key"
                  value={p.key}
                  onChange={(e) => updateMeta(i, "key", e.target.value)}
                  style={{ width: 180 }}
                />
                <input
                  placeholder="Value"
                  value={p.value}
                  onChange={(e) => updateMeta(i, "value", e.target.value)}
                  style={{ flex: 1 }}
                />
                <button
                  type="button"
                  onClick={() => removeMeta(i)}
                  style={{
                    background: "var(--surface-container-highest)",
                    color: "var(--error)",
                    borderRadius: "var(--radius)",
                    padding: "4px 10px",
                    fontSize: 16,
                  }}
                >
                  &times;
                </button>
              </div>
            ))}
          </div>

          {error && (
            <div style={{ color: "var(--error)", fontSize: 13, marginBottom: 16 }}>{error}</div>
          )}

          <button
            type="submit"
            disabled={submitting || !collection.trim() || !content.trim()}
            style={{
              background: "linear-gradient(135deg, var(--primary), var(--primary-container))",
              color: "var(--on-primary-fixed)",
              borderRadius: "var(--radius)",
              padding: "12px 32px",
              fontWeight: 600,
              fontSize: 15,
              opacity: submitting || !collection.trim() || !content.trim() ? 0.5 : 1,
              transition: "opacity 0.15s",
            }}
          >
            {submitting ? "Storing..." : "Store Context"}
          </button>
        </form>
      </div>
    </div>
  );
}
