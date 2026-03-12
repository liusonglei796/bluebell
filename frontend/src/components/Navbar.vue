<template>
  <nav class="bg-white shadow">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="flex justify-between h-16">
        <div class="flex">
          <router-link to="/" class="flex-shrink-0 flex items-center">
            <span class="text-xl font-bold text-indigo-600">Bluebell</span>
          </router-link>
        </div>
        <div class="flex items-center space-x-4">
          <template v-if="authStore.token">
            <span class="text-gray-700">Welcome, {{ authStore.user?.username }}</span>
            <button @click="logout" class="text-sm font-medium text-gray-500 hover:text-gray-900">
              Logout
            </button>
            <router-link to="/create-post" class="bg-indigo-600 text-white px-3 py-2 rounded-md text-sm font-medium hover:bg-indigo-700">
              New Post
            </router-link>
          </template>
          <template v-else>
            <router-link to="/login" class="text-sm font-medium text-gray-500 hover:text-gray-900">
              Login
            </router-link>
            <router-link to="/signup" class="bg-indigo-600 text-white px-3 py-2 rounded-md text-sm font-medium hover:bg-indigo-700">
              Sign up
            </router-link>
          </template>
        </div>
      </div>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { useAuthStore } from '../store/auth';
import { useRouter } from 'vue-router';

const authStore = useAuthStore();
const router = useRouter();

const logout = () => {
  authStore.clearAuth();
  router.push('/login');
};
</script>