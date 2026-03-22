import axios from 'axios';

const NEXT_PUBLIC_GATEWAY_URL = process.env.NEXT_PUBLIC_GATEWAY_URL || 'http://localhost:30000';

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

// Response interceptor to handle 401s (optional: redirect to login if session expires)
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      console.warn('Unauthorized request. Possible session expiry.');
      // Optionally clear token or trigger re-auth
    }
    return Promise.reject(error);
  }
);
