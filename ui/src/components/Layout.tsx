import { NavLink, Outlet, Link, useNavigate } from "react-router-dom";
import { useApi } from "../hooks/useApi";
import { useAuth } from "../hooks/useAuth";
import { listCollections, healthCheck } from "../api";
import type { CSSProperties } from "react";

const sidebarStyle: CSSProperties = {
  width: 240,
  minHeight: "100vh",
  background: "var(--surface-container-low)",
  display: "flex",
  flexDirection: "column",
  padding: "24px 0",
  flexShrink: 0,
};

const logoStyle: CSSProperties = {
  fontFamily: "var(--font-headline)",
  fontSize: 20,
  fontWeight: 800,
  color: "var(--primary)",
  padding: "0 20px",
  marginBottom: 32,
  letterSpacing: "-0.02em",
};

const navStyle: CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: 2,
  padding: "0 12px",
};

const baseLinkStyle: CSSProperties = {
  display: "flex",
  alignItems: "center",
  gap: 10,
  padding: "10px 12px",
  borderRadius: "var(--radius)",
  fontSize: 14,
  fontWeight: 500,
  color: "var(--on-surface-variant)",
  textDecoration: "none",
  transition: "all 0.15s",
};

const activeLinkExtra: CSSProperties = {
  background: "var(--surface-container-high)",
  color: "var(--primary)",
  boxShadow: "inset 3px 0 0 var(--primary)",
};

const sectionTitleStyle: CSSProperties = {
  fontFamily: "var(--font-label)",
  fontSize: 11,
  fontWeight: 500,
  letterSpacing: "0.05em",
  textTransform: "uppercase" as const,
  color: "var(--outline)",
  padding: "0 20px",
  marginTop: 28,
  marginBottom: 8,
};

const collectionItemStyle: CSSProperties = {
  fontSize: 13,
  color: "var(--on-surface-variant)",
  padding: "6px 20px",
  cursor: "default",
  transition: "color 0.15s",
};

const userInfoStyle: CSSProperties = {
  display: "flex",
  alignItems: "center",
  gap: 10,
  padding: "16px 20px",
  marginTop: "auto",
  borderTop: "1px solid var(--outline-variant)",
};

const avatarStyle: CSSProperties = {
  width: 32,
  height: 32,
  borderRadius: "50%",
  background: "linear-gradient(135deg, var(--primary), var(--primary-container))",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  color: "var(--on-primary)",
  fontWeight: 700,
  fontSize: 14,
  fontFamily: "var(--font-headline)",
  flexShrink: 0,
};

const logoutBtnStyle: CSSProperties = {
  background: "none",
  border: "none",
  color: "var(--on-surface-variant)",
  cursor: "pointer",
  padding: 4,
  display: "flex",
  alignItems: "center",
};

function NavIcon({ d }: { d: string }) {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d={d} />
    </svg>
  );
}

export default function Layout() {
  const { data: collections } = useApi(listCollections);
  const { data: health } = useApi(healthCheck);
  const { user, authRequired, logout } = useAuth();
  const navigate = useNavigate();

  // Open mode (no users yet): show Users so the first admin can be created without logging in.
  const showUsersNav = !authRequired || user?.role === "admin";
  const showStressNav = showUsersNav;
  const showTokensNav = !!user;
  const showSettingsSection = showUsersNav || showTokensNav;

  async function handleLogout() {
    await logout();
    navigate("/login");
  }

  return (
    <div style={{ display: "flex", minHeight: "100vh" }}>
      <aside style={sidebarStyle}>
        <div style={logoStyle}>LaightDB</div>
        <nav style={navStyle}>
          <NavLink
            to="/"
            end
            style={({ isActive }) => ({ ...baseLinkStyle, ...(isActive ? activeLinkExtra : {}) })}
          >
            <NavIcon d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-4 0h4" />
            Dashboard
          </NavLink>
          <NavLink
            to="/search"
            style={({ isActive }) => ({ ...baseLinkStyle, ...(isActive ? activeLinkExtra : {}) })}
          >
            <NavIcon d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            Search
          </NavLink>
          <NavLink
            to="/store"
            style={({ isActive }) => ({ ...baseLinkStyle, ...(isActive ? activeLinkExtra : {}) })}
          >
            <NavIcon d="M12 4v16m8-8H4" />
            Store Context
          </NavLink>
          <NavLink
            to="/collections"
            style={({ isActive }) => ({ ...baseLinkStyle, ...(isActive ? activeLinkExtra : {}) })}
          >
            <NavIcon d="M4 6h16M4 12h16M4 18h7" />
            Collections
          </NavLink>
          <NavLink
            to="/system"
            style={({ isActive }) => ({ ...baseLinkStyle, ...(isActive ? activeLinkExtra : {}) })}
          >
            <NavIcon d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
            System
          </NavLink>
          {showStressNav && (
            <NavLink
              to="/stress"
              style={({ isActive }) => ({ ...baseLinkStyle, ...(isActive ? activeLinkExtra : {}) })}
            >
              <NavIcon d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" />
              Stress test
            </NavLink>
          )}

          {showSettingsSection && <div style={sectionTitleStyle}>Settings</div>}
          {showUsersNav && (
            <NavLink
              to="/settings/users"
              style={({ isActive }) => ({ ...baseLinkStyle, ...(isActive ? activeLinkExtra : {}) })}
            >
              <NavIcon d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2M9 11a4 4 0 100-8 4 4 0 000 8M23 21v-2a4 4 0 00-3-3.87M16 3.13a4 4 0 010 7.75" />
              Users
            </NavLink>
          )}
          {showTokensNav && (
            <NavLink
              to="/settings/tokens"
              style={({ isActive }) => ({ ...baseLinkStyle, ...(isActive ? activeLinkExtra : {}) })}
            >
              <NavIcon d="M15 7h3a5 5 0 010 10h-3m-6 0H6a5 5 0 010-10h3M8 12h8" />
              API Tokens
            </NavLink>
          )}
        </nav>

        <div style={sectionTitleStyle}>Collections</div>
        <div style={{ flex: 1, overflow: "auto" }}>
          {collections?.map((c) => (
            <Link
              key={c}
              to={`/collections/${encodeURIComponent(c)}`}
              style={{ ...collectionItemStyle, display: "block", color: "var(--on-surface-variant)", textDecoration: "none" }}
            >
              {c}
            </Link>
          ))}
          {collections?.length === 0 && (
            <div style={{ ...collectionItemStyle, color: "var(--outline)", fontStyle: "italic" }}>
              No collections yet
            </div>
          )}
        </div>

        {user && authRequired && (
          <div style={userInfoStyle}>
            <div style={avatarStyle}>{user.username[0].toUpperCase()}</div>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ fontSize: 13, fontWeight: 600, color: "var(--on-surface)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                {user.username}
              </div>
              <div style={{ fontSize: 11, color: "var(--on-surface-variant)", fontFamily: "var(--font-label)" }}>
                {user.role}
              </div>
            </div>
            <button style={logoutBtnStyle} onClick={handleLogout} title="Sign out">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M9 21H5a2 2 0 01-2-2V5a2 2 0 012-2h4M16 17l5-5-5-5M21 12H9" />
              </svg>
            </button>
          </div>
        )}
      </aside>

      <div style={{ flex: 1, display: "flex", flexDirection: "column", minHeight: "100vh" }}>
        <header
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            padding: "16px 32px",
            background: "var(--surface-container-lowest)",
          }}
        >
          <span style={{ fontFamily: "var(--font-headline)", fontWeight: 700, fontSize: 16, color: "var(--on-surface)" }}>
            LaightDB
          </span>
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: 8,
              fontFamily: "var(--font-label)",
              fontSize: 12,
              fontWeight: 500,
            }}
          >
            <span
              style={{
                width: 8,
                height: 8,
                borderRadius: "50%",
                background: health?.status === "ok" ? "#4ade80" : "var(--error)",
              }}
            />
            <span style={{ color: health?.status === "ok" ? "#4ade80" : "var(--error)" }}>
              {health ? (health.status === "ok" ? "Healthy" : "Unhealthy") : "Checking..."}
            </span>
          </div>
        </header>
        <main style={{ flex: 1, padding: 32, overflow: "auto" }}>
          <Outlet />
        </main>
      </div>
    </div>
  );
}
