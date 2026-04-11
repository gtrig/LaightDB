import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from "react";
import type { UserInfo } from "../types";
import { getAuthStatus, getMe, logout as apiLogout, login as apiLogin } from "../api";

interface AuthContextValue {
  user: UserInfo | null;
  authRequired: boolean;
  loading: boolean;
  login: (username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refresh: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<UserInfo | null>(null);
  const [authRequired, setAuthRequired] = useState(false);
  const [loading, setLoading] = useState(true);

  const loadAuth = useCallback(async () => {
    try {
      const status = await getAuthStatus();
      setAuthRequired(status.auth_required);
      if (status.auth_required) {
        try {
          const me = await getMe();
          setUser(me.user);
        } catch {
          setUser(null);
        }
      }
    } catch {
      // Server unreachable
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { loadAuth(); }, [loadAuth]);

  const login = useCallback(async (username: string, password: string) => {
    const result = await apiLogin(username, password);
    setUser(result.user);
    setAuthRequired(true);
  }, []);

  const logout = useCallback(async () => {
    await apiLogout();
    setUser(null);
  }, []);

  return (
    <AuthContext value={{ user, authRequired, loading, login, logout, refresh: loadAuth }}>
      {children}
    </AuthContext>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
