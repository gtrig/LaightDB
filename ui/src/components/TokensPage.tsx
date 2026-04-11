import { useState, type FormEvent, type CSSProperties } from "react";
import { useApi } from "../hooks/useApi";
import { listTokens, createToken, revokeToken } from "../api";
import type { UserRole } from "../types";

/**
 * Template aligned with Stitch “API Tokens Settings”
 * projects/8721321444339603457/screens/b412253939a049ccaf9f9bd9d71c9093
 */
const shellStyle: CSSProperties = {
  background: "var(--surface-container-low)",
  borderRadius: "var(--radius)",
  padding: 28,
  maxWidth: 1100,
};

const headingStyle: CSSProperties = {
  fontFamily: "var(--font-headline)",
  fontSize: 24,
  fontWeight: 700,
  color: "var(--on-surface)",
  marginBottom: 4,
};

const subStyle: CSSProperties = {
  color: "var(--on-surface-variant)",
  fontSize: 13,
  marginBottom: 24,
};

const createBtnStyle: CSSProperties = {
  padding: "10px 20px",
  background: "linear-gradient(135deg, var(--primary), var(--primary-container))",
  color: "var(--on-primary)",
  border: "none",
  borderRadius: "var(--radius)",
  fontSize: 14,
  fontWeight: 600,
  cursor: "pointer",
};

const badgeStyle = (role: UserRole): CSSProperties => ({
  display: "inline-block",
  padding: "3px 10px",
  borderRadius: "var(--radius-full)",
  fontSize: 12,
  fontWeight: 600,
  fontFamily: "var(--font-label)",
  background: role === "admin" ? "rgba(107, 216, 203, 0.15)" : "var(--surface-container-highest)",
  color: role === "admin" ? "var(--primary)" : "var(--on-surface-variant)",
});

const statusBadge = (active: boolean): CSSProperties => ({
  display: "inline-block",
  padding: "3px 10px",
  borderRadius: "var(--radius-full)",
  fontSize: 12,
  fontWeight: 600,
  fontFamily: "var(--font-label)",
  background: active ? "rgba(74, 222, 128, 0.12)" : "rgba(255, 180, 171, 0.12)",
  color: active ? "#4ade80" : "var(--error)",
});

const actionBtnStyle: CSSProperties = {
  padding: "6px 12px",
  fontSize: 12,
  borderRadius: "var(--radius)",
  cursor: "pointer",
  border: "none",
  fontFamily: "var(--font-body)",
};

const overlayStyle: CSSProperties = {
  position: "fixed",
  inset: 0,
  background: "rgba(0,0,0,0.6)",
  backdropFilter: "blur(8px)",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  zIndex: 1000,
};

const modalStyle: CSSProperties = {
  background: "var(--surface-container)",
  borderRadius: "var(--radius)",
  padding: 32,
  width: "100%",
  maxWidth: 480,
  boxShadow: "0 24px 80px rgba(0, 0, 0, 0.45)",
};

const modalTitleStyle: CSSProperties = {
  fontFamily: "var(--font-headline)",
  fontSize: 18,
  fontWeight: 700,
  color: "var(--on-surface)",
  marginBottom: 24,
};

const labelStyle: CSSProperties = {
  display: "block",
  fontFamily: "var(--font-label)",
  fontSize: 11,
  fontWeight: 500,
  letterSpacing: "0.05em",
  textTransform: "uppercase",
  color: "var(--on-surface-variant)",
  marginBottom: 6,
};

const inputStyle: CSSProperties = {
  width: "100%",
  padding: "10px 14px",
  marginBottom: 16,
};

const tokenDisplayStyle: CSSProperties = {
  background: "var(--surface-container-lowest)",
  borderRadius: "var(--radius)",
  padding: 16,
  marginBottom: 20,
};

const tokenValueStyle: CSSProperties = {
  fontFamily: "var(--font-label)",
  fontSize: 13,
  color: "var(--primary)",
  wordBreak: "break-all",
  lineHeight: 1.6,
};

export default function TokensPage() {
  const { data: tokens, loading, error, refetch } = useApi(listTokens);
  const [showModal, setShowModal] = useState(false);
  const [form, setForm] = useState({ name: "", role: "readonly" as UserRole });
  const [newToken, setNewToken] = useState<string | null>(null);
  const [formError, setFormError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [copied, setCopied] = useState(false);

  async function handleCreate(e: FormEvent) {
    e.preventDefault();
    setFormError("");
    setSubmitting(true);
    try {
      const result = await createToken(form.name, form.role);
      setNewToken(result.token);
      setForm({ name: "", role: "readonly" });
      refetch();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Failed");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleRevoke(id: string) {
    try {
      await revokeToken(id);
      refetch();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed");
    }
  }

  function handleCopy() {
    if (newToken) {
      void navigator.clipboard.writeText(newToken);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }

  if (loading) {
    return (
      <div style={shellStyle}>
        <div style={{ color: "var(--on-surface-variant)" }}>Loading tokens…</div>
      </div>
    );
  }

  if (error) {
    return (
      <div style={shellStyle}>
        <h1 style={headingStyle}>API Tokens</h1>
        <p style={{ color: "var(--error)" }}>{error}</p>
      </div>
    );
  }

  return (
    <div style={shellStyle}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 20, flexWrap: "wrap", gap: 16 }}>
        <div>
          <h1 style={headingStyle}>API Tokens</h1>
          <p style={subStyle}>
            Use tokens for REST and MCP over HTTP (<code style={{ fontFamily: "var(--font-label)" }}>Authorization: Bearer …</code>). The full secret is shown only once when you create a token.
          </p>
        </div>
        <button
          type="button"
          style={createBtnStyle}
          onClick={() => {
            setShowModal(true);
            setNewToken(null);
            setFormError("");
          }}
        >
          Create Token
        </button>
      </div>

      <div style={{ overflowX: "auto" }}>
        <table>
          <thead>
            <tr style={{ background: "var(--surface-container-lowest)" }}>
              <th>Name</th>
              <th>Prefix</th>
              <th>Role</th>
              <th>Created</th>
              <th>Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {tokens?.map((t, i) => (
              <tr
                key={t.id}
                style={{
                  background: i % 2 === 0 ? "var(--surface-container-low)" : "var(--surface-container-lowest)",
                }}
              >
                <td style={{ fontWeight: 600 }}>{t.name}</td>
                <td>
                  <code style={{ fontFamily: "var(--font-label)", fontSize: 13, color: "var(--on-surface-variant)" }}>
                    {t.prefix}…
                  </code>
                </td>
                <td>
                  <span style={badgeStyle(t.role)}>{t.role}</span>
                </td>
                <td style={{ color: "var(--on-surface-variant)", fontSize: 13 }}>
                  {new Date(t.created_at).toLocaleString()}
                </td>
                <td>
                  <span style={statusBadge(t.active)}>{t.active ? "Active" : "Revoked"}</span>
                </td>
                <td>
                  {t.active && (
                    <button
                      type="button"
                      style={{ ...actionBtnStyle, background: "rgba(255,180,171,0.15)", color: "var(--error)" }}
                      onClick={() => handleRevoke(t.id)}
                    >
                      Revoke
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {tokens?.length === 0 && (
        <div style={{ textAlign: "center", color: "var(--outline)", padding: 40 }}>No API tokens yet</div>
      )}

      {showModal && (
        <div style={overlayStyle} onClick={(e) => { if (e.target === e.currentTarget) { setShowModal(false); setNewToken(null); } }} role="presentation">
          <div style={modalStyle}>
            {newToken ? (
              <>
                <div style={modalTitleStyle}>Token created</div>
                <div style={tokenDisplayStyle}>
                  <div style={tokenValueStyle}>{newToken}</div>
                </div>
                <div style={{ display: "flex", gap: 12, marginBottom: 16, flexWrap: "wrap" }}>
                  <button type="button" style={createBtnStyle} onClick={handleCopy}>
                    {copied ? "Copied" : "Copy to clipboard"}
                  </button>
                </div>
                <div style={{ color: "var(--tertiary)", fontSize: 13, fontWeight: 500, marginBottom: 24 }}>
                  This token will not be shown again. Store it in a password manager or secret store.
                </div>
                <button
                  type="button"
                  style={{ ...actionBtnStyle, background: "var(--surface-container-high)", color: "var(--on-surface-variant)", padding: "10px 20px" }}
                  onClick={() => { setShowModal(false); setNewToken(null); }}
                >
                  Done
                </button>
              </>
            ) : (
              <form onSubmit={handleCreate}>
                <div style={modalTitleStyle}>Create API token</div>
                <label style={labelStyle} htmlFor="token-name">Name</label>
                <input
                  id="token-name"
                  style={inputStyle}
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  placeholder="e.g. CI, MCP client"
                  required
                  autoFocus
                />
                <label style={labelStyle} htmlFor="token-role">Role</label>
                <select
                  id="token-role"
                  style={{ ...inputStyle, marginBottom: 24 }}
                  value={form.role}
                  onChange={(e) => setForm({ ...form, role: e.target.value as UserRole })}
                >
                  <option value="readonly">Read only</option>
                  <option value="admin">Admin</option>
                </select>
                {formError && <div style={{ color: "var(--error)", fontSize: 13, marginBottom: 16 }}>{formError}</div>}
                <div style={{ display: "flex", gap: 12, justifyContent: "flex-end" }}>
                  <button
                    type="button"
                    style={{ ...actionBtnStyle, background: "var(--surface-container-high)", color: "var(--on-surface-variant)", padding: "10px 20px" }}
                    onClick={() => { setShowModal(false); setNewToken(null); }}
                  >
                    Cancel
                  </button>
                  <button type="submit" style={createBtnStyle} disabled={submitting}>
                    {submitting ? "Creating…" : "Create"}
                  </button>
                </div>
              </form>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
