import { defineStore } from 'pinia';
import { appwriteService } from '../services/appwriteService';

interface User {
  user_id: string;
  username: string;
  role: number; // 1=普通用户, 2=管理员
}

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('token') || '',
    refreshToken: localStorage.getItem('refresh_token') || '',
    user: null as User | null,
  }),
  actions: {
    setAuth(token: string, refreshToken: string, username: string, userId: string, role: number) {
      this.token = token;
      this.refreshToken = refreshToken;
      this.user = { username, user_id: userId, role };
      localStorage.setItem('token', token);
      localStorage.setItem('refresh_token', refreshToken);
      localStorage.setItem('username', username);
      localStorage.setItem('user_id', userId);
      localStorage.setItem('role', String(role));
    },
    async clearAuth() {
      this.token = '';
      this.refreshToken = '';
      this.user = null;
      localStorage.removeItem('token');
      localStorage.removeItem('refresh_token');
      localStorage.removeItem('username');
      localStorage.removeItem('user_id');
      localStorage.removeItem('role');
      await appwriteService.logout();
    },
    updateTokens(token: string, refreshToken: string) {
      this.token = token;
      this.refreshToken = refreshToken;
      localStorage.setItem('token', token);
      localStorage.setItem('refresh_token', refreshToken);
    },
    async init() {
      const username = localStorage.getItem('username');
      const userId = localStorage.getItem('user_id');
      const role = localStorage.getItem('role');
      if (username && userId) {
        this.user = { username, user_id: userId, role: role ? parseInt(role, 10) : 1 };
      }
      
      // 尝试从 Appwrite 自动同步真实 Session 登录态
      try {
        const currentUser = await appwriteService.getCurrentUser();
        if (currentUser) {
          const validUsername = currentUser.username || 'User';
          this.user = {
            user_id: currentUser.user_id,
            username: validUsername,
            role: currentUser.role,
          };
          this.token = currentUser.user_id;
          localStorage.setItem('username', validUsername);
          localStorage.setItem('user_id', currentUser.user_id);
        }
      } catch {
        // 未登录或 Session 过期
      }
    },
    // 检查是否为管理员或已登录创作者
    isAdmin(): boolean {
      return this.user?.role === 2 || this.user !== null;
    }
  },
});