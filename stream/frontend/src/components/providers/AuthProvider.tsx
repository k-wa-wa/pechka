'use client';

import React, { createContext, useContext, useEffect, useState } from 'react';
import { apiClient } from '@/lib/api-client';

interface User {
  id: string;
  email: string;
  displayName: string;
  roles: string[];
  groups: string[];
  permissions: string[];
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  isLoading: boolean;
  logout: () => void;
  refreshSession: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider = ({ children }: { children: React.ReactNode }) => {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const fetchMe = async (authToken: string) => {
    try {
      const res = await apiClient.get('/api/v1/auth/me', {
        headers: { Authorization: `Bearer ${authToken}` }
      });
      setUser(res.data);
    } catch (err) {
      console.error('Failed to fetch user info:', err);
      // If /me fails with a token, the token might be invalid
      localStorage.removeItem('app_jwt');
      setToken(null);
    }
  };

  const refreshSession = async () => {
    try {
      setIsLoading(true);
      // Call /session to exchange Cloudflare/Proxy cookie for App JWT
      const res = await apiClient.get('/api/v1/auth/session');
      const newToken = res.data.access_token;
      
      localStorage.setItem('app_jwt', newToken);
      setToken(newToken);
      await fetchMe(newToken);
    } catch (err) {
      console.error('Failed to establish auth session from gateway cookie:', err);
      localStorage.removeItem('app_jwt');
      setToken(null);
      setUser(null);
    } finally {
      setIsLoading(false);
    }
  };

  const logout = () => {
    localStorage.removeItem('app_jwt');
    setToken(null);
    setUser(null);
    // Redirect to the gateway-level logout endpoint.
    // Dev:  dev-proxy handles /cdn-cgi/access/logout → clears cookie → redirects to login
    // Prod: Cloudflare Access handles the same path natively
    window.location.href = '/api/v1/auth/logout';
  };

  useEffect(() => {
    const initAuth = async () => {
      const storedToken = localStorage.getItem('app_jwt');
      if (storedToken) {
        setToken(storedToken);
        await fetchMe(storedToken);
        setIsLoading(false);
      } else {
        await refreshSession();
      }
    };

    initAuth();
  }, []);

  return (
    <AuthContext.Provider value={{ user, token, isLoading, logout, refreshSession }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
