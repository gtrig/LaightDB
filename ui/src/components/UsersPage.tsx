import { useState, type FormEvent, type CSSProperties } from "react";
import { useApi } from "../hooks/useApi";
import { useAuth } from "../hooks/useAuth";
import { listUsers, createUser, deleteUser, changeRole } from "../api";
import type { UserRole } from "../types";

/**
 * Template aligned with Stitch “Admin Settings” (Users)
 * projects/8721321444339603457/screens/907711debb45466594fc8af847693cdc
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

const addBtnStyle: CSSProperties = {
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
  maxWidth: 420,
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

export default function UsersPage() {
  const { user: current } = useAuth();
  const { data: users, loading, error, refetch } = useApi(listUsers);
  const [showModal, setShowModal] = useState(false);
  const [form, setForm] = useState({ username: "", password: "", role: "readonly" as UserRole });
  const [formError, setFormError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);

  async function handleCreate(e: FormEvent) {
    e.preventDefault();
    setFormError("");
    setSubmitting(true);
    try {
      await createUser(form.username, form.password, form.role);
      setShowModal(false);
      setForm({ username: "", password: "", role: "readonly" });
      refetch();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Failed");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(id: string) {
    try {
      await deleteUser(id);
      setConfirmDelete(null);
      refetch();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed");
    }
  }

  async function handleRoleChange(id: string, role: UserRole) {
    try {
      await changeRole(id, role);
      refetch();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed");
    }
  }

  if (loading) {
    return (
      <div style={shellStyle}>
        <div style={{ color: "var(--on-surface-variant)" }}>Loading users…</div>
      </div>
    );
  }

  if (error) {
    return (
      <div style={shellStyle}>
        <h1 style={headingStyle}>Users</h1>
        <p style={{ color: "var(--error)" }}>{error}</p>
      </div>
    );
  }

  return (
    <div style={shellStyle}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 20, flexWrap: "wrap", gap: 16 }}>
        <div>
          <h1 style={headingStyle}>Users</h1>
          <p style={subStyle}>Create accounts, adjust roles, or remove access. Deleting a user revokes their API tokens and sessions.</p>
        </div>
        <button type="button" style={addBtnStyle} onClick={() => setShowModal(true)}>
          Add User
        </button>
      </div>

      <div style={{ overflowX: "auto" }}>
        <table>
          <thead>
            <tr style={{ background: "var(--surface-container-lowest)" }}>
              <th>Username</th>
              <th>Role</th>
              <th>Created</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {users?.map((u, i) => {
              const isSelf = current?.id === u.id;
              return (
                <tr
                  key={u.id}
                  style={{
                    background: i % 2 === 0 ? "var(--surface-container-low)" : "var(--surface-container-lowest)",
                  }}
                >
                  <td style={{ fontWeight: 600 }}>{u.username}{isSelf ? " (you)" : ""}</td>
                  <td>
                    <span style={badgeStyle(u.role)}>{u.role}</span>
                  </td>
                  <td style={{ color: "var(--on-surface-variant)", fontSize: 13 }}>
                    {new Date(u.created_at).toLocaleString()}
                  </td>
                  <td style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                    <select
                      value={u.role}
                      onChange={(e) => handleRoleChange(u.id, e.target.value as UserRole)}
                      disabled={isSelf}
                      title={isSelf ? "Use another admin to change your role" : "Change role"}
                      style={{
                        ...actionBtnStyle,
                        background: "var(--surface-container-high)",
                        color: "var(--on-surface)",
                        padding: "6px 8px",
                        opacity: isSelf ? 0.5 : 1,
                      }}
                    >
                      <option value="admin">admin</option>
                      <option value="readonly">readonly</option>
                    </select>
                    {!isSelf && (
                      <button
                        type="button"
                        style={{ ...actionBtnStyle, background: "rgba(255,180,171,0.15)", color: "var(--error)" }}
                        onClick={() => setConfirmDelete(u.id)}
                      >
                        Delete
                      </button>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {users?.length === 0 && (
        <div style={{ textAlign: "center", color: "var(--outline)", padding: 40 }}>No users yet</div>
      )}

      {showModal && (
        <div style={overlayStyle} onClick={(e) => { if (e.target === e.currentTarget) setShowModal(false); }} role="presentation">
          <form style={modalStyle} onSubmit={handleCreate}>
            <div style={modalTitleStyle}>Add User</div>
            <label style={labelStyle} htmlFor="new-username">Username</label>
            <input
              id="new-username"
              style={inputStyle}
              value={form.username}
              onChange={(e) => setForm({ ...form, username: e.target.value })}
              placeholder="Username"
              required
              autoFocus
            />
            <label style={labelStyle} htmlFor="new-password">Password</label>
            <input
              id="new-password"
              style={inputStyle}
              type="password"
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
              placeholder="Password"
              required
            />
            <label style={labelStyle} htmlFor="new-role">Role</label>
            <select
              id="new-role"
              style={{ ...inputStyle, marginBottom: 24 }}
              value={form.role}
              onChange={(e) => setForm({ ...form, role: e.target.value as UserRole })}
            >
              <option value="readonly">Read Only</option>
              <option value="admin">Admin</option>
            </select>
            {formError && <div style={{ color: "var(--error)", fontSize: 13, marginBottom: 16 }}>{formError}</div>}
            <div style={{ display: "flex", gap: 12, justifyContent: "flex-end" }}>
              <button
                type="button"
                style={{ ...actionBtnStyle, background: "var(--surface-container-high)", color: "var(--on-surface-variant)", padding: "10px 20px" }}
                onClick={() => setShowModal(false)}
              >
                Cancel
              </button>
              <button type="submit" style={addBtnStyle} disabled={submitting}>
                {submitting ? "Creating…" : "Create"}
              </button>
            </div>
          </form>
        </div>
      )}

      {confirmDelete && (
        <div style={overlayStyle} onClick={(e) => { if (e.target === e.currentTarget) setConfirmDelete(null); }} role="presentation">
          <div style={modalStyle}>
            <div style={modalTitleStyle}>Delete User</div>
            <p style={{ color: "var(--on-surface-variant)", marginBottom: 24 }}>
              Are you sure? This will also revoke all their API tokens and sessions.
            </p>
            <div style={{ display: "flex", gap: 12, justifyContent: "flex-end" }}>
              <button
                type="button"
                style={{ ...actionBtnStyle, background: "var(--surface-container-high)", color: "var(--on-surface-variant)", padding: "10px 20px" }}
                onClick={() => setConfirmDelete(null)}
              >
                Cancel
              </button>
              <button
                type="button"
                style={{ ...actionBtnStyle, background: "var(--error-container)", color: "var(--error)", padding: "10px 20px" }}
                onClick={() => handleDelete(confirmDelete)}
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
