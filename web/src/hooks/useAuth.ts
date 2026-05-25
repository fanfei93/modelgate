import { useState, useEffect, useCallback } from 'react';
import type { UserBrief } from '../types/api';
import * as authApi from '../api/auth';

export function useAuth() {
  const [user, setUser] = useState<UserBrief | null>(() => {
    const stored = localStorage.getItem('modelgate_user');
    return stored ? JSON.parse(stored) : null;
  });
  const [loading, setLoading] = useState(true);

  const isAuthenticated = !!user;

  useEffect(() => {
    const token = localStorage.getItem('modelgate_token');
    if (token) {
      authApi.getMe()
        .then((res) => {
          if (res.data.user) {
            setUser(res.data.user);
            localStorage.setItem('modelgate_user', JSON.stringify(res.data.user));
          } else {
            logoutLocal();
          }
        })
        .catch(() => logoutLocal())
        .finally(() => setLoading(false));
    } else {
      setLoading(false);
    }
  }, []);

  const loginAction = useCallback(async (username: string, password: string) => {
    const res = await authApi.login(username, password);
    const { token, user: userData } = res.data;
    if (token && userData) {
      localStorage.setItem('modelgate_token', token);
      localStorage.setItem('modelgate_user', JSON.stringify(userData));
      setUser(userData);
    }
    return res.data;
  }, []);

  const registerAction = useCallback(async (username: string, email: string, password: string, code: string) => {
    const res = await authApi.register(username, email, password, code);
    const { token, user: userData } = res.data;
    if (token && userData) {
      localStorage.setItem('modelgate_token', token);
      localStorage.setItem('modelgate_user', JSON.stringify(userData));
      setUser(userData);
    }
    return res.data;
  }, []);

  const logout = useCallback(() => {
    logoutLocal();
    setUser(null);
  }, []);

  return { user, loading, isAuthenticated, login: loginAction, register: registerAction, logout };
}

function logoutLocal() {
  localStorage.removeItem('modelgate_token');
  localStorage.removeItem('modelgate_user');
}
