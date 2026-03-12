import { createRouter, createWebHistory } from 'vue-router';
import type { RouteRecordRaw } from 'vue-router';
import { useAuthStore } from '../store/auth';

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'Home',
    component: () => import('../pages/Home.vue'),
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('../pages/Login.vue'),
  },
  {
    path: '/signup',
    name: 'Signup',
    component: () => import('../pages/Signup.vue'),
  },
  {
    path: '/community/:id',
    name: 'CommunityDetail',
    component: () => import('../pages/CommunityDetail.vue'),
  },
  {
    path: '/post/:id',
    name: 'PostDetail',
    component: () => import('../pages/PostDetail.vue'),
  },
  {
    path: '/create-post',
    name: 'CreatePost',
    component: () => import('../pages/CreatePost.vue'),
    meta: { requiresAuth: true },
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach((to, _from, next) => {
  const authStore = useAuthStore();
  if (to.meta.requiresAuth && !authStore.token) {
    next({ name: 'Login' });
  } else {
    next();
  }
});

export default router;