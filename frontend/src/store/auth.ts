import { defineStore } from 'pinia';

interface User {
  user_id: string;
  username: string;
}

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('token') || '',
    refreshToken: localStorage.getItem('refresh_token') || '',
    user: null as User | null,
  }),
  actions: {
    setAuth(token: string, refreshToken: string, username: string, userId: string) {
      this.token = token;
      this.refreshToken = refreshToken;
      this.user = { username, user_id: userId };
      localStorage.setItem('token', token);
      localStorage.setItem('refresh_token', refreshToken);
      localStorage.setItem('username', username);
      localStorage.setItem('user_id', userId);
    },
    clearAuth() {
      this.token = '';
      this.refreshToken = '';
      this.user = null;
      localStorage.removeItem('token');
      localStorage.removeItem('refresh_token');
      localStorage.removeItem('username');
      localStorage.removeItem('user_id');
    },
    updateTokens(token: string, refreshToken: string) {
      this.token = token;
      this.refreshToken = refreshToken;
      localStorage.setItem('token', token);
      localStorage.setItem('refresh_token', refreshToken);
    },
    init() {
      const username = localStorage.getItem('username');
      const userId = localStorage.getItem('user_id');
      if (username && userId) {
        this.user = { username, user_id: userId };
      }
    }
  },
});