import axios from 'axios';
import { useAuthStore } from '../store/auth';
import router from '../router';

const request = axios.create({
  baseURL: '/api/v1',
  timeout: 5000,
});

request.interceptors.request.use(
  (config) => {
    const authStore = useAuthStore();
    if (authStore.token) {
      config.headers.Authorization = `Bearer ${authStore.token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

request.interceptors.response.use(
  (response) => {
    return response.data;
  },
  async (error) => {
    const originalRequest = error.config;
    if (error.response && error.response.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;
      const authStore = useAuthStore();
      
      try {
        const res = await axios.post('/api/v1/refresh_token', null, {
          headers: {
            Authorization: `Bearer ${authStore.refreshToken}`
          }
        });
        
        // Assuming the response structure is { code: 1000, msg: "success", data: { access_token, refresh_token } }
        if (res.data.code === 1000) {
          authStore.updateTokens(res.data.data.access_token, res.data.data.refresh_token);
          originalRequest.headers.Authorization = `Bearer ${res.data.data.access_token}`;
          return request(originalRequest);
        }
      } catch (refreshError) {
        authStore.clearAuth();
        router.push('/login');
        return Promise.reject(refreshError);
      }
    }
    return Promise.reject(error);
  }
);

export default request;