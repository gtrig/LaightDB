import { useState, type CSSProperties, type FormEvent } from "react";
import { useApi } from "../hooks/useApi";
import { getStressQueries, runStress } from "../api";
import type { StressReport } from "../types";

const cardStyle: CSSProperties = {
  background: "var(--surface-container-low)",
  borderRadius: "var(--radius)",
  padding: "20px 24px",
  marginBottom: 16,
  borderLeft: "2px solid var(--primary)",
};

const labelStyle: CSSProperties = {
  fontFamily: "var(--font-label)",
  fontSize: 11,
  letterSpacing: "0.05em",
  color: "var(--outline)",
  marginBottom: 6,
  display: "block",
};

const inputRowStyle: CSSProperties = {
  display: "grid",
  gridTemplateColumns: "repeat(auto-fill, minmax(160px, 1fr))",
  gap: 16,
  marginBottom: 16,
};

const MAX_WRITES = 50000;
const MAX_SEARCHES = 500000;
const MAX_CONCURRENCY = 128;

function fmtNs(ns: number): string {
  if (!Number.isFinite(ns) || ns < 0) return "—";
  const ms = ns / 1e6;
  if (ms < 10) return `${ms.toFixed(2)} ms`;
  if (ms < 1000) return `${Math.round(ms)} ms`;
  return `${(ms / 1000).toFixed(2)} s`;
}

function PhaseTable({ title, p }: { title: string; p: StressReport["writes"] }) {
  return (
    <div style={cardStyle}>
      <div style={{ ...labelStyle, marginBottom: 12 }}>{title}</div>
      <div style={{ fontSize: 14, lineHeight: 1.8 }}>
        <div>
          <strong>Requested / OK / errors:</strong> {p.requested} / {p.ok} / {p.errors}
        </div>
        <div>
          <strong>Wall time:</strong> {fmtNs(p.wall)}
        </div>
        <div>
          <strong>p50 / p95 / p99:</strong> {fmtNs(p.p50)} / {fmtNs(p.p95)} / {fmtNs(p.p99)}
        </div>
        <div>
          <strong>Throughput:</strong> {p.ops_per_sec.toFixed(2)} ops/s
        </div>
      </div>
    </div>
  );
}

export default function StressPage() {
  const { data: queries, loading: queriesLoading, error: queriesError } = useApi(getStressQueries);
  const [collection, setCollection] = useState("stress");
  const [writes, setWrites] = useState(50);
  const [writeConcurrency, setWriteConcurrency] = useState(4);
  const [searches, setSearches] = useState(200);
  const [searchConcurrency, setSearchConcurrency] = useState(8);
  const [topK, setTopK] = useState(10);
  const [detail, setDetail] = useState("summary");
  const [running, setRunning] = useState(false);
  const [runError, setRunError] = useState<string | null>(null);
  const [report, setReport] = useState<StressReport | null>(null);
  const [queriesOpen, setQueriesOpen] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setRunning(true);
    setRunError(null);
    setReport(null);
    try {
      const r = await runStress({
        collection: collection.trim() || "stress",
        writes: Math.min(Math.max(0, writes), MAX_WRITES),
        write_concurrency: Math.min(Math.max(1, writeConcurrency), MAX_CONCURRENCY),
        searches: Math.min(Math.max(0, searches), MAX_SEARCHES),
        search_concurrency: Math.min(Math.max(1, searchConcurrency), MAX_CONCURRENCY),
        top_k: Math.max(1, topK),
        detail,
      });
      setReport(r);
    } catch (err) {
      setRunError(err instanceof Error ? err.message : "Run failed");
    } finally {
      setRunning(false);
    }
  }

  return (
    <div style={{ maxWidth: 720 }}>
      <h1 style={{ fontFamily: "var(--font-headline)", fontSize: 24, fontWeight: 700, marginBottom: 8 }}>
        Stress test
      </h1>
      <p style={{ color: "var(--on-surface-variant)", marginBottom: 24, fontSize: 14, lineHeight: 1.5 }}>
        Runs an in-process benchmark on the server: writes synthetic documents into a collection, then runs a fixed set of hybrid
        searches (same queries as <code style={{ fontSize: 13 }}>laightdb-stress</code>). Metrics exclude HTTP overhead. Requires
        admin when authentication is enabled; in open mode anyone can run this.
      </p>

      <div style={cardStyle}>
        <div style={{ ...labelStyle, marginBottom: 8 }}>STANDARD SEARCH QUERIES</div>
        {queriesLoading && <div style={{ color: "var(--outline)", fontSize: 14 }}>Loading…</div>}
        {queriesError && <div style={{ color: "var(--error)", fontSize: 14 }}>{queriesError}</div>}
        {queries && queries.length > 0 && (
          <>
            <button
              type="button"
              onClick={() => setQueriesOpen(!queriesOpen)}
              style={{
                background: "var(--surface-container-highest)",
                color: "var(--primary)",
                borderRadius: "var(--radius)",
                padding: "6px 14px",
                fontSize: 13,
                marginBottom: queriesOpen ? 12 : 0,
              }}
            >
              {queriesOpen ? "Hide" : "Show"} {queries.length} queries
            </button>
            {queriesOpen && (
              <ul style={{ margin: 0, paddingLeft: 20, fontSize: 13, color: "var(--on-surface-variant)", lineHeight: 1.6 }}>
                {queries.map((q) => (
                  <li key={q}>{q}</li>
                ))}
              </ul>
            )}
          </>
        )}
      </div>

      <form onSubmit={onSubmit} style={cardStyle}>
        <div style={{ ...labelStyle, marginBottom: 12 }}>WORKLOAD</div>
        <div style={inputRowStyle}>
          <div>
            <label style={labelStyle}>Collection</label>
            <input
              value={collection}
              onChange={(e) => setCollection(e.target.value)}
              style={{ width: "100%", padding: "8px 10px", borderRadius: "var(--radius)", border: "1px solid var(--outline-variant)" }}
            />
          </div>
          <div>
            <label style={labelStyle}>Detail</label>
            <select
              value={detail}
              onChange={(e) => setDetail(e.target.value)}
              style={{ width: "100%", padding: "8px 10px", borderRadius: "var(--radius)", border: "1px solid var(--outline-variant)" }}
            >
              <option value="metadata">metadata</option>
              <option value="summary">summary</option>
              <option value="full">full</option>
            </select>
          </div>
          <div>
            <label style={labelStyle}>Top K</label>
            <input
              type="number"
              min={1}
              value={topK}
              onChange={(e) => setTopK(Number(e.target.value))}
              style={{ width: "100%", padding: "8px 10px", borderRadius: "var(--radius)", border: "1px solid var(--outline-variant)" }}
            />
          </div>
        </div>
        <div style={inputRowStyle}>
          <div>
            <label style={labelStyle}>Writes (max {MAX_WRITES.toLocaleString()})</label>
            <input
              type="number"
              min={0}
              max={MAX_WRITES}
              value={writes}
              onChange={(e) => setWrites(Number(e.target.value))}
              style={{ width: "100%", padding: "8px 10px", borderRadius: "var(--radius)", border: "1px solid var(--outline-variant)" }}
            />
          </div>
          <div>
            <label style={labelStyle}>Write concurrency (max {MAX_CONCURRENCY})</label>
            <input
              type="number"
              min={1}
              max={MAX_CONCURRENCY}
              value={writeConcurrency}
              onChange={(e) => setWriteConcurrency(Number(e.target.value))}
              style={{ width: "100%", padding: "8px 10px", borderRadius: "var(--radius)", border: "1px solid var(--outline-variant)" }}
            />
          </div>
        </div>
        <div style={inputRowStyle}>
          <div>
            <label style={labelStyle}>Searches (max {MAX_SEARCHES.toLocaleString()})</label>
            <input
              type="number"
              min={0}
              max={MAX_SEARCHES}
              value={searches}
              onChange={(e) => setSearches(Number(e.target.value))}
              style={{ width: "100%", padding: "8px 10px", borderRadius: "var(--radius)", border: "1px solid var(--outline-variant)" }}
            />
          </div>
          <div>
            <label style={labelStyle}>Search concurrency (max {MAX_CONCURRENCY})</label>
            <input
              type="number"
              min={1}
              max={MAX_CONCURRENCY}
              value={searchConcurrency}
              onChange={(e) => setSearchConcurrency(Number(e.target.value))}
              style={{ width: "100%", padding: "8px 10px", borderRadius: "var(--radius)", border: "1px solid var(--outline-variant)" }}
            />
          </div>
        </div>
        <button
          type="submit"
          disabled={running}
          style={{
            background: "linear-gradient(135deg, var(--primary), var(--primary-container))",
            color: "var(--on-primary-fixed)",
            borderRadius: "var(--radius)",
            padding: "10px 20px",
            fontWeight: 600,
            fontSize: 14,
            opacity: running ? 0.6 : 1,
          }}
        >
          {running ? "Running…" : "Run stress test"}
        </button>
        {runError && (
          <div style={{ marginTop: 12, fontSize: 14, color: "var(--error)" }}>{runError}</div>
        )}
      </form>

      {report && (
        <>
          <div style={{ ...cardStyle, borderLeftColor: "var(--primary-container)" }}>
            <div style={{ ...labelStyle, marginBottom: 8 }}>SUMMARY</div>
            <div style={{ fontSize: 14 }}>
              <strong>Collection:</strong> {report.collection}
              <br />
              <strong>Mode:</strong> {report.base_url}
              <br />
              <strong>Total wall:</strong> {fmtNs(report.total_wall)}
            </div>
          </div>
          <PhaseTable title="WRITES" p={report.writes} />
          <PhaseTable title="SEARCHES" p={report.searches} />
        </>
      )}
    </div>
  );
}
