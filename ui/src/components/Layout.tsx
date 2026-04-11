import { NavLink, Outlet } from "react-router-dom";
import { useApi } from "../hooks/useApi";
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
        </nav>

        <div style={sectionTitleStyle}>Collections</div>
        <div style={{ flex: 1, overflow: "auto" }}>
          {collections?.map((c) => (
            <div key={c} style={collectionItemStyle}>{c}</div>
          ))}
          {collections?.length === 0 && (
            <div style={{ ...collectionItemStyle, color: "var(--outline)", fontStyle: "italic" }}>
              No collections yet
            </div>
          )}
        </div>
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
