import { Fragment, useState, type CSSProperties } from "react";
import { useApi } from "../hooks/useApi";
import { listAuditCalls } from "../api";
import type { CallLogEntry } from "../types";

const cardStyle: CSSProperties = {
  background: "var(--surface-container-low)",
  borderRadius: "var(--radius)",
  overflow: "hidden",
};

const mono: CSSProperties = {
  fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
  fontSize: 12,
  lineHeight: 1.55,
  letterSpacing: "0.01em",
  whiteSpace: "pre-wrap",
  wordBreak: "break-word",
  margin: 0,
  padding: "14px 16px",
  background: "var(--surface-container-highest)",
  borderRadius: "var(--radius)",
  borderLeft: "3px solid var(--primary)",
  maxHeight: "min(70vh, 480px)",
  overflow: "auto",
};

/** Pretty-print JSON when the whole string parses; otherwise show raw text. */
function formatLogPayload(raw: string | undefined): { text: string; isJSON: boolean } {
  if (raw == null || raw.trim() === "") {
    return { text: "—", isJSON: false };
  }
  const t = raw.trim();
  try {
    const v = JSON.parse(t);
    return { text: JSON.stringify(v, null, 2), isJSON: true };
  } catch {
    return { text: raw, isJSON: false };
  }
}

function FormattedLogPayload({ title, raw }: { title: string; raw?: string }) {
  const { text, isJSON } = formatLogPayload(raw);
  return (
    <div style={{ marginBottom: title === "Response" ? 0 : 16 }}>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          marginBottom: 8,
          gap: 8,
        }}
      >
        <span style={{ fontSize: 11, fontWeight: 600, letterSpacing: "0.06em", color: "var(--outline)", textTransform: "uppercase" }}>
          {title}
        </span>
        {isJSON && (
          <span
            style={{
              fontSize: 10,
              fontFamily: "var(--font-label)",
              fontWeight: 600,
              color: "var(--primary)",
              background: "var(--surface-container-high)",
              padding: "2px 8px",
              borderRadius: 4,
            }}
          >
            JSON
          </span>
        )}
      </div>
      <pre style={mono}>{text}</pre>
    </div>
  );
}

function whatLabel(c: CallLogEntry): string {
  return c.tool ?? "(tool)";
}

function statusLabel(c: CallLogEntry): string {
  return c.ok === false ? "error" : "ok";
}

export default function CallLogsPage() {
  const { data: calls, loading, error } = useApi(() => listAuditCalls(200));
  const [expanded, setExpanded] = useState<string | null>(null);

  return (
    <div>
      <h1
        style={{
          fontFamily: "var(--font-headline)",
          fontSize: 24,
          fontWeight: 700,
          marginBottom: 8,
        }}
      >
        MCP call log
      </h1>
      <p style={{ color: "var(--on-surface-variant)", marginBottom: 24, fontSize: 14 }}>
        Recent MCP tool calls (inputs and responses are truncated on the server).
      </p>

      {error && (
        <div style={{ color: "var(--error)", marginBottom: 16 }}>{error}</div>
      )}

      <div style={cardStyle}>
        <table>
          <thead>
            <tr>
              <th>Time</th>
              <th>Who</th>
              <th>Tool</th>
              <th>Status</th>
              <th style={{ textAlign: "right" }}>ms</th>
            </tr>
          </thead>
          <tbody>
            {loading && !calls?.length && (
              <tr>
                <td colSpan={5} style={{ textAlign: "center", color: "var(--outline)", padding: 32 }}>
                  Loading…
                </td>
              </tr>
            )}
            {!loading && calls && calls.length === 0 && (
              <tr>
                <td colSpan={5} style={{ textAlign: "center", color: "var(--outline)", padding: 32 }}>
                  No calls recorded yet
                </td>
              </tr>
            )}
            {calls?.map((c) => (
              <Fragment key={c.id}>
                <tr
                  style={{ cursor: "pointer" }}
                  onClick={() => setExpanded((prev) => (prev === c.id ? null : c.id))}
                >
                  <td style={{ fontSize: 12, color: "var(--on-surface-variant)" }}>
                    {new Date(c.ts).toLocaleString()}
                  </td>
                  <td style={{ fontSize: 12 }}>
                    {c.username ? (
                      <>
                        <span style={{ fontWeight: 600 }}>{c.username}</span>
                        {c.user_id && (
                          <span style={{ color: "var(--outline)", marginLeft: 6, fontSize: 11 }}>
                            {c.user_id.substring(0, 8)}…
                          </span>
                        )}
                      </>
                    ) : (
                      <span style={{ color: "var(--outline)" }}>—</span>
                    )}
                  </td>
                  <td style={{ fontSize: 13, maxWidth: 360 }}>{whatLabel(c)}</td>
                  <td style={{ fontSize: 12 }}>{statusLabel(c)}</td>
                  <td style={{ textAlign: "right", fontSize: 12 }}>{c.duration_ms}</td>
                </tr>
                {expanded === c.id && (
                  <tr>
                    <td colSpan={5} style={{ padding: "12px 16px 20px", verticalAlign: "top" }}>
                      {c.query && (
                        <div style={{ marginBottom: 16, fontSize: 12, color: "var(--on-surface-variant)" }}>
                          <strong>Query:</strong> {c.query}
                        </div>
                      )}
                      <FormattedLogPayload title="Request" raw={c.request} />
                      <FormattedLogPayload title="Response" raw={c.response} />
                    </td>
                  </tr>
                )}
              </Fragment>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
