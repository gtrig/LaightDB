import { lazy, Suspense } from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import { useAuth } from "./hooks/useAuth";
import Layout from "./components/Layout";
import Dashboard from "./components/Dashboard";
import SearchPanel from "./components/SearchPanel";
import StoreForm from "./components/StoreForm";
import ContextDetail from "./components/ContextDetail";
import CollectionsPage from "./components/CollectionsPage";
import CollectionBrowse from "./components/CollectionBrowse";
import SystemPage from "./components/SystemPage";
import StressPage from "./components/StressPage";
import LoginPage from "./components/LoginPage";
import UsersPage from "./components/UsersPage";
import TokensPage from "./components/TokensPage";
import CallLogsPage from "./components/CallLogsPage";

const StorageExplorer3D = lazy(() => import("./components/StorageExplorer3D"));

function RequireAuth({ children }: { children: React.ReactNode }) {
  const { user, authRequired, loading } = useAuth();
  if (loading) {
    return (
      <div style={{ minHeight: "100vh", display: "flex", alignItems: "center", justifyContent: "center", color: "var(--on-surface-variant)" }}>
        Loading…
      </div>
    );
  }
  if (authRequired && !user) return <Navigate to="/login" replace />;
  return children;
}

/** User management: admins always; in open mode (no users yet) anyone can open to bootstrap the first account. */
function RequireAdminOrBootstrap({ children }: { children: React.ReactNode }) {
  const { user, authRequired, loading } = useAuth();
  if (loading) {
    return (
      <div style={{ minHeight: "100vh", display: "flex", alignItems: "center", justifyContent: "center", color: "var(--on-surface-variant)" }}>
        Loading…
      </div>
    );
  }
  if (!authRequired) return children;
  if (!user || user.role !== "admin") return <Navigate to="/" replace />;
  return children;
}

/** API tokens require a logged-in user; in open mode, send users to bootstrap first. */
function RequireUserForTokens({ children }: { children: React.ReactNode }) {
  const { user, authRequired, loading } = useAuth();
  if (loading) {
    return (
      <div style={{ minHeight: "100vh", display: "flex", alignItems: "center", justifyContent: "center", color: "var(--on-surface-variant)" }}>
        Loading…
      </div>
    );
  }
  if (user) return children;
  if (authRequired) return <Navigate to="/login" replace />;
  return <Navigate to="/settings/users" replace />;
}

/** Call log UI: admin only when auth is active; hidden in open mode (API returns 403). */
function RequireAdminWhenAuth({ children }: { children: React.ReactNode }) {
  const { user, authRequired, loading } = useAuth();
  if (loading) {
    return (
      <div style={{ minHeight: "100vh", display: "flex", alignItems: "center", justifyContent: "center", color: "var(--on-surface-variant)" }}>
        Loading…
      </div>
    );
  }
  if (!authRequired) return <Navigate to="/" replace />;
  if (!user || user.role !== "admin") return <Navigate to="/" replace />;
  return children;
}

export default function App() {
  const { user, authRequired, loading } = useAuth();

  if (loading) {
    return (
      <div style={{ minHeight: "100vh", display: "flex", alignItems: "center", justifyContent: "center", color: "var(--on-surface-variant)", fontFamily: "var(--font-body)" }}>
        Loading…
      </div>
    );
  }

  const loginElement = authRequired && !user ? <LoginPage /> : <Navigate to="/" replace />;

  return (
    <Routes>
      <Route path="/login" element={loginElement} />
      <Route
        element={
          <RequireAuth>
            <Layout />
          </RequireAuth>
        }
      >
        <Route index element={<Dashboard />} />
        <Route path="search" element={<SearchPanel />} />
        <Route path="store" element={<StoreForm />} />
        <Route path="collections" element={<CollectionsPage />} />
        <Route path="collections/:name" element={<CollectionBrowse />} />
        <Route path="system" element={<SystemPage />} />
        <Route
          path="stress"
          element={
            <RequireAdminOrBootstrap>
              <StressPage />
            </RequireAdminOrBootstrap>
          }
        />
        <Route
          path="audit"
          element={
            <RequireAdminWhenAuth>
              <CallLogsPage />
            </RequireAdminWhenAuth>
          }
        />
        <Route
          path="explorer"
          element={
            <Suspense
              fallback={
                <div style={{ minHeight: 240, display: "flex", alignItems: "center", justifyContent: "center", color: "var(--on-surface-variant)" }}>
                  Loading…
                </div>
              }
            >
              <StorageExplorer3D />
            </Suspense>
          }
        />
        <Route path="contexts/:id" element={<ContextDetail />} />
        <Route
          path="settings/users"
          element={
            <RequireAdminOrBootstrap>
              <UsersPage />
            </RequireAdminOrBootstrap>
          }
        />
        <Route
          path="settings/tokens"
          element={
            <RequireUserForTokens>
              <TokensPage />
            </RequireUserForTokens>
          }
        />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}
