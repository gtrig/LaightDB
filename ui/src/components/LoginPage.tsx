import { useState, type FormEvent, type CSSProperties } from "react";
import { useAuth } from "../hooks/useAuth";

/**
 * Visual template aligned with Stitch screen “Login (Dark Mode)”
 * projects/8721321444339603457/screens/6a68e2ae174142ffbc89033a1a1fb8ce
 */
const pageStyle: CSSProperties = {
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  minHeight: "100vh",
  background: "var(--stitch-surface)",
  padding: 24,
};

const cardStyle: CSSProperties = {
  width: "100%",
  maxWidth: 420,
  padding: "40px 36px",
  background: "var(--stitch-card)",
  borderRadius: "var(--radius)",
  boxShadow: "0 24px 80px rgba(0, 0, 0, 0.45)",
};

const logoStyle: CSSProperties = {
  fontFamily: "var(--font-headline)",
  fontSize: 28,
  fontWeight: 800,
  color: "var(--primary)",
  textAlign: "center",
  letterSpacing: "-0.02em",
  marginBottom: 8,
};

const taglineStyle: CSSProperties = {
  textAlign: "center",
  color: "var(--on-surface-variant)",
  fontSize: 13,
  lineHeight: 1.5,
  marginBottom: 28,
  maxWidth: 320,
  marginLeft: "auto",
  marginRight: "auto",
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
  padding: "12px 14px",
  background: "var(--stitch-input)",
  color: "var(--on-surface)",
  border: "1px solid rgba(61, 73, 71, 0.35)",
  borderRadius: "var(--radius)",
  fontSize: 14,
  marginBottom: 18,
};

const buttonStyle: CSSProperties = {
  width: "100%",
  padding: "14px 24px",
  background: "linear-gradient(135deg, var(--primary) 0%, var(--stitch-gradient-end) 100%)",
  color: "var(--on-primary)",
  border: "none",
  borderRadius: "var(--radius-full)",
  fontSize: 15,
  fontWeight: 600,
  cursor: "pointer",
  fontFamily: "var(--font-body)",
  marginTop: 8,
};

const errorStyle: CSSProperties = {
  background: "rgba(147, 0, 10, 0.35)",
  color: "var(--error)",
  fontSize: 13,
  textAlign: "center",
  marginTop: 16,
  padding: "10px 12px",
  borderRadius: "var(--radius)",
};

export default function LoginPage() {
  const { login } = useAuth();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await login(username, password);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={pageStyle}>
      <form style={cardStyle} onSubmit={handleSubmit} aria-labelledby="login-heading">
        <div id="login-heading" style={logoStyle}>
          LaightDB
        </div>
        <p style={taglineStyle}>
          Sign in to manage context, API tokens, and users. Same session as the REST API (HTTP-only cookie).
        </p>

        <label style={labelStyle} htmlFor="login-username">
          Username
        </label>
        <input
          id="login-username"
          style={inputStyle}
          type="text"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          placeholder="Enter username"
          autoComplete="username"
          autoFocus
          required
        />

        <label style={labelStyle} htmlFor="login-password">
          Password
        </label>
        <input
          id="login-password"
          style={inputStyle}
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="Enter password"
          autoComplete="current-password"
          required
        />

        <button style={buttonStyle} type="submit" disabled={loading}>
          {loading ? "Signing in…" : "Sign In"}
        </button>

        {error && <div style={errorStyle} role="alert">{error}</div>}
      </form>
    </div>
  );
}
