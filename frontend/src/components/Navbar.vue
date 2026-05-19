<template>
  <nav class="fixed top-4 left-0 right-0 z-50 px-4">
    <div class="max-w-7xl mx-auto glass rounded-2xl shadow-lg">
      <div class="px-4 sm:px-6 lg:px-8">
        <div class="flex justify-between h-16">
          <div class="flex">
            <router-link to="/" class="flex-shrink-0 flex items-center">
              <span class="text-xl font-extrabold bg-clip-text text-transparent bg-[image:var(--accent-gradient)]">Bluebell</span>
            </router-link>
            
            <!-- Search Bar -->
            <div class="hidden sm:ml-6 sm:flex sm:items-center">
              <form @submit.prevent="handleSearch" class="relative">
                <input 
                  v-model="searchKeyword"
                  type="text" 
                  class="block w-full pl-3 pr-10 py-1.5 border border-white/20 rounded-xl leading-5 bg-white/30 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/50 sm:text-sm backdrop-blur-sm transition-all" 
                  placeholder="Search Bluebell"
                >
                <button type="submit" class="absolute inset-y-0 right-0 pr-3 flex items-center">
                  <svg class="h-4 w-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path></svg>
                </button>
              </form>
            </div>
          </div>
          <div class="flex items-center space-x-4">
            <template v-if="authStore.token">
              <span class="text-sm font-medium text-gray-700">Welcome, {{ authStore.user?.username }}</span>
              <router-link
                v-if="authStore.isAdmin()"
                to="/create-community"
                class="text-sm font-medium text-indigo-600 hover:text-indigo-800 transition-colors"
              >
                New Community
              </router-link>
              <button @click="logout" class="text-sm font-medium text-gray-500 hover:text-gray-900 transition-colors">
                Logout
              </button>
              <router-link to="/create-post" class="bg-[image:var(--accent-gradient)] text-white px-4 py-2 rounded-xl text-sm font-bold shadow-md hover:shadow-lg transition-all active:scale-95">
                New Post
              </router-link>
            </template>
            <template v-else>
              <router-link to="/login" class="text-sm font-medium text-gray-500 hover:text-gray-900 transition-colors">
                Login
              </router-link>
              <router-link to="/signup" class="bg-[image:var(--accent-gradient)] text-white px-4 py-2 rounded-xl text-sm font-bold shadow-md hover:shadow-lg transition-all active:scale-95">
                Sign up
              </router-link>
            </template>
          </div>
        </div>
      </div>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useAuthStore } from '../store/auth';
import { useRouter } from 'vue-router';

const authStore = useAuthStore();
const router = useRouter();
const searchKeyword = ref('');

const handleSearch = () => {
  if (searchKeyword.value.trim()) {
    router.push({
      path: '/search',
      query: { q: searchKeyword.value.trim() }
    });
    searchKeyword.value = '';
  }
};

const logout = () => {
  authStore.clearAuth();
  router.push('/login');
};
</script>