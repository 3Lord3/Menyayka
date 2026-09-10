import axios, { AxiosError } from 'axios';

import { env } from '@shared/config/env';
import { tokenStorage, userStorage } from '@shared/lib/storage';

const API_BASE_URL = `${env.apiBaseUrl}/api/v1`;

export const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: { 'Content-Type': 'application/json' },
});

apiClient.interceptors.request.use((config) => {
  const token = tokenStorage.get();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    // logging out with an expired token returns 401 — this is a normal logout scenario,
    // handled locally by useLogout (clear + SPA redirect), without a full re-login
    const isLogoutCall = error.config?.url === '/auth/logout';
    // the token expired or was revoked by the server — clear storage (token and user, as logout())
    // and send to login
    if (error.response?.status === 401 && !isLogoutCall && window.location.pathname !== '/login') {
      tokenStorage.remove();
      userStorage.remove();
      window.location.assign('/login');
    }
    return Promise.reject(error);
  },
);
