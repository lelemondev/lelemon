'use client';

import { createContext, useContext, useState, useEffect, ReactNode, useCallback } from 'react';
import { useRouter } from 'next/navigation';

const API_URL = process.env.NEXT_PUBLIC_API_URL || '';

export interface User {
  id: string;
  email: string;
  name: string;
  createdAt: string;
}

interface AuthContextType {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  loginWithToken: (token: string) => Promise<void>;
  register: (email: string, password: string, name: string) => Promise<void>;
  logout: () => void;
  refreshToken: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

const USER_KEY = 'lelemon_user';

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const router = useRouter();

  const saveUser = useCallback((newUser: User) => {
    try {
      localStorage.setItem(USER_KEY, JSON.stringify(newUser));
    } catch {
      // localStorage may not be available
    }
    setUser(newUser);
  }, []);

  const clearAuth = useCallback(() => {
    try {
      localStorage.removeItem(USER_KEY);
      localStorage.removeItem('lelemon_current_project');
    } catch {
      // localStorage may not be available
    }
    setUser(null);
  }, []);

  // On mount: check if we have a valid session by calling /auth/me
  useEffect(() => {
    // First try cached user for instant UI
    try {
      const savedUser = localStorage.getItem(USER_KEY);
      if (savedUser) {
        setUser(JSON.parse(savedUser));
      }
    } catch {
      // ignore parse errors
    }

    // Then verify session is still valid via cookie
    fetch(`${API_URL}/api/v1/auth/me`, { credentials: 'include' })
      .then(async (res) => {
        if (res.ok) {
          const userData = await res.json();
          saveUser(userData);
        } else {
          clearAuth();
        }
      })
      .catch(() => {
        // Offline or server down — keep cached user
      })
      .finally(() => {
        setIsLoading(false);
      });
  }, [saveUser, clearAuth]);

  const login = useCallback(async (email: string, password: string) => {
    const response = await fetch(`${API_URL}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include', // receive httpOnly cookie
      body: JSON.stringify({ email, password }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Login failed');
    }

    const data = await response.json();
    saveUser(data.user);
    router.push('/dashboard');
  }, [router, saveUser]);

  const loginWithToken = useCallback(async (token: string) => {
    // Used after OAuth exchange — token is already in cookie, but we also got it from exchange
    const response = await fetch(`${API_URL}/api/v1/auth/me`, {
      headers: { 'Authorization': `Bearer ${token}` },
    });

    if (!response.ok) {
      throw new Error('Failed to authenticate with token');
    }

    const userData = await response.json();
    saveUser(userData);
  }, [saveUser]);

  const register = useCallback(async (email: string, password: string, name: string) => {
    const response = await fetch(`${API_URL}/api/v1/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ email, password, name }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Registration failed');
    }

    const data = await response.json();
    saveUser(data.user);
    router.push('/dashboard');
  }, [router, saveUser]);

  const logout = useCallback(async () => {
    try {
      await fetch(`${API_URL}/api/v1/auth/logout`, {
        method: 'POST',
        credentials: 'include',
      });
    } catch {
      // Best effort — clear local state regardless
    }
    clearAuth();
    router.push('/login');
  }, [router, clearAuth]);

  const refreshToken = useCallback(async () => {
    try {
      const response = await fetch(`${API_URL}/api/v1/auth/refresh`, {
        method: 'POST',
        credentials: 'include',
      });

      if (response.ok) {
        const data = await response.json();
        saveUser(data.user);
      } else {
        clearAuth();
      }
    } catch {
      clearAuth();
    }
  }, [saveUser, clearAuth]);

  return (
    <AuthContext.Provider
      value={{
        user,
        isLoading,
        isAuthenticated: !!user,
        login,
        loginWithToken,
        register,
        logout,
        refreshToken,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}

// Helper to get API URL
export function getApiUrl(): string {
  return API_URL;
}
