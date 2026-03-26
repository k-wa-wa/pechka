import axios from 'axios';

// 空文字にすることでブラウザの現在オリジンへの相対リクエストになる
const NEXT_PUBLIC_GATEWAY_URL = process.env.NEXT_PUBLIC_GATEWAY_URL || '';

export const apiClient = axios.create({
  baseURL: NEXT_PUBLIC_GATEWAY_URL,
  withCredentials: true, // Needed for dev-proxy cookies
});

// Request interceptor to add the App JWT to headers
apiClient.interceptors.request.use((config) => {
  if (typeof window !== 'undefined') {
    const token = localStorage.getItem('app_jwt');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
  }
  return config;
});

// Response interceptor: on 401, clear the stored JWT so AuthProvider re-fetches it.
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      if (typeof window !== 'undefined') {
        localStorage.removeItem('app_jwt');
      }
    }
    return Promise.reject(error);
  }
);
