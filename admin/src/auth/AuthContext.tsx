import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { api, ApiError } from '../api/client';
import type { Admin } from '../api/types';

interface AuthState {
  loading: boolean;
  admin: Admin | null;
  bootstrapped: boolean | null;
  refresh: () => Promise<void>;
  setAdmin: (admin: Admin | null) => void;
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [loading, setLoading] = useState(true);
  const [admin, setAdmin] = useState<Admin | null>(null);
  const [bootstrapped, setBootstrapped] = useState<boolean | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const status = await api.get<{ bootstrapped: boolean }>('/auth/status');
      setBootstrapped(status.bootstrapped);
      if (status.bootstrapped) {
        try {
          const me = await api.get<{ admin: Admin }>('/me');
          setAdmin(me.admin);
        } catch (err) {
          if (err instanceof ApiError && err.status === 401) {
            setAdmin(null);
          } else {
            throw err;
          }
        }
      } else {
        setAdmin(null);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh().catch(() => setLoading(false));
  }, [refresh]);

  const value = useMemo(
    () => ({ loading, admin, bootstrapped, refresh, setAdmin }),
    [loading, admin, bootstrapped, refresh],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error('useAuth must be used inside AuthProvider');
  }
  return ctx;
}
