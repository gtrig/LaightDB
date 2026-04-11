import { useState, type CSSProperties, type FormEvent } from "react";
import { searchContexts, getContext } from "../api";
import { useApi } from "../hooks/useApi";
import { listCollections } from "../api";
import type { ContextEntry } from "../types";
import ContentTypeBadge from "./ContentTypeBadge";
import CollectionBadge from "./CollectionBadge";
import { useNavigate } from "react-router-dom";

interface Hit {
  id: string;
  score: number;
  entry?: ContextEntry;
}

const filterRowStyle: CSSProperties = {
  display: "flex",
  gap: 12,
  alignItems: "flex-end",
  flexWrap: "wrap",
  marginBottom: 24,
};

const labelStyle: CSSProperties = {
  fontFamily: "var(--font-label)",
  fontSize: 11,
  fontWeight: 500,
  letterSpacing: "0.05em",
  textTransform: "uppercase",
  color: "var(--on-surface-variant)",
  marginBottom: 6,
};

const resultCardStyle: CSSProperties = {
  background: "var(--surface-container-low)",
  borderRadius: "var(--radius)",
  padding: "16px 20px",
  cursor: "pointer",
  transition: "background 0.15s",
};

export default function SearchPanel() {
  const [query, setQuery] = useState("");
  const [collection, setCollection] = useState("");
  const [topK, setTopK] = useState(10);
  const [metaFilters, setMetaFilters] = useState<{ key: string; value: string }[]>([]);
  const [hits, setHits] = useState<Hit[]>([]);
  const [searching, setSearching] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { data: collections } = useApi(listCollections);
  const navigate = useNavigate();

  async function handleSearch(e: FormEvent) {
    e.preventDefault();
    if (!query.trim()) return;
    setSearching(true);
    setError(null);
    try {
      const filters: Record<string, string> = {};
      for (const f of metaFilters) {
        if (f.key.trim() && f.value.trim()) filters[f.key] = f.value;
      }
      const results = await searchContexts({
        query,
        collection: collection || undefined,
        filters: Object.keys(filters).length > 0 ? filters : undefined,
        top_k: topK,
      });
      const enriched = await Promise.all(
        results.map(async (r) => {
          const entry = await getContext(r.ID, "summary").catch(() => undefined);
          return { id: r.ID, score: r.Score, entry };
        })
      );
      setHits(enriched);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Search failed");
    } finally {
      setSearching(false);
    }
  }

  function addFilter() {
    setMetaFilters([...metaFilters, { key: "", value: "" }]);
  }

  function removeFilter(i: number) {
    setMetaFilters(metaFilters.filter((_, idx) => idx !== i));
  }

  function updateFilter(i: number, field: "key" | "value", val: string) {
    const updated = [...metaFilters];
    updated[i] = { ...updated[i], [field]: val };
    setMetaFilters(updated);
  }

  return (
    <div>
      <h1 style={{ fontFamily: "var(--font-headline)", fontSize: 24, fontWeight: 700, marginBottom: 24 }}>
        Search
      </h1>

      <form onSubmit={handleSearch}>
        <div style={{ position: "relative", marginBottom: 16 }}>
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="var(--outline)"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            style={{ position: "absolute", left: 14, top: "50%", transform: "translateY(-50%)" }}
          >
            <path d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search contexts..."
            style={{ width: "100%", paddingLeft: 42, fontSize: 16, padding: "14px 14px 14px 42px" }}
          />
        </div>

        <div style={filterRowStyle}>
          <div>
            <div style={labelStyle}>Collection</div>
            <select value={collection} onChange={(e) => setCollection(e.target.value)} style={{ minWidth: 160 }}>
              <option value="">All collections</option>
              {collections?.map((c) => (
                <option key={c} value={c}>{c}</option>
              ))}
            </select>
          </div>

          <div>
            <div style={labelStyle}>Results (top_k)</div>
            <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
              <input
                type="range"
                min={1}
                max={50}
                value={topK}
                onChange={(e) => setTopK(Number(e.target.value))}
                style={{
                  width: 120,
                  background: "transparent",
                  accentColor: "var(--primary)",
                  padding: 0,
                }}
              />
              <span style={{ fontFamily: "var(--font-label)", fontSize: 13, color: "var(--on-surface-variant)", minWidth: 24 }}>
                {topK}
              </span>
            </div>
          </div>

          <button
            type="submit"
            disabled={searching}
            style={{
              background: "linear-gradient(135deg, var(--primary), var(--primary-container))",
              color: "var(--on-primary-fixed)",
              borderRadius: "var(--radius)",
              padding: "10px 24px",
              fontWeight: 600,
              fontSize: 14,
              opacity: searching ? 0.6 : 1,
            }}
          >
            {searching ? "Searching..." : "Search"}
          </button>
        </div>

        <div style={{ marginBottom: 16 }}>
          <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 8 }}>
            <span style={labelStyle}>Metadata Filters</span>
            <button
              type="button"
              onClick={addFilter}
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
          {metaFilters.map((f, i) => (
            <div key={i} style={{ display: "flex", gap: 8, marginBottom: 8 }}>
              <input
                placeholder="Key"
                value={f.key}
                onChange={(e) => updateFilter(i, "key", e.target.value)}
                style={{ width: 160 }}
              />
              <input
                placeholder="Value"
                value={f.value}
                onChange={(e) => updateFilter(i, "value", e.target.value)}
                style={{ width: 240 }}
              />
              <button
                type="button"
                onClick={() => removeFilter(i)}
                style={{
                  background: "var(--surface-container-highest)",
                  color: "var(--error)",
                  borderRadius: "var(--radius)",
                  padding: "4px 10px",
                  fontSize: 14,
                }}
              >
                &times;
              </button>
            </div>
          ))}
        </div>
      </form>

      {error && (
        <div style={{ color: "var(--error)", marginBottom: 16, fontSize: 13 }}>{error}</div>
      )}

      <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
        {hits.map((hit) => (
          <div
            key={hit.id}
            style={resultCardStyle}
            onClick={() => navigate(`/contexts/${hit.id}`)}
            onMouseEnter={(e) => { e.currentTarget.style.background = "var(--surface-container-high)"; }}
            onMouseLeave={(e) => { e.currentTarget.style.background = "var(--surface-container-low)"; }}
          >
            <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 8 }}>
              <div
                style={{
                  fontFamily: "var(--font-label)",
                  fontSize: 14,
                  fontWeight: 600,
                  color: "var(--primary)",
                  minWidth: 48,
                }}
              >
                {(hit.score * 100).toFixed(0)}%
              </div>
              <div
                style={{
                  flex: 1,
                  height: 4,
                  background: "var(--surface-container-highest)",
                  borderRadius: 2,
                  overflow: "hidden",
                }}
              >
                <div
                  style={{
                    width: `${hit.score * 100}%`,
                    height: "100%",
                    background: "linear-gradient(90deg, var(--primary), var(--primary-container))",
                    borderRadius: 2,
                  }}
                />
              </div>
            </div>
            <div style={{ display: "flex", alignItems: "center", gap: 10, flexWrap: "wrap" }}>
              <span style={{ fontFamily: "var(--font-label)", fontSize: 12, color: "var(--primary)" }}>
                {hit.id.substring(0, 12)}...
              </span>
              {hit.entry && <CollectionBadge name={hit.entry.collection} />}
              {hit.entry && <ContentTypeBadge type={hit.entry.content_type} />}
            </div>
            {hit.entry?.summary && (
              <p style={{ marginTop: 8, fontSize: 13, color: "var(--on-surface-variant)", lineHeight: 1.5 }}>
                {hit.entry.summary.substring(0, 200)}
                {hit.entry.summary.length > 200 ? "..." : ""}
              </p>
            )}
          </div>
        ))}
        {hits.length === 0 && !searching && query && (
          <div style={{ textAlign: "center", color: "var(--outline)", padding: 48 }}>
            No results found
          </div>
        )}
      </div>
    </div>
  );
}
