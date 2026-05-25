<template>
  <nav class="fixed top-4 left-0 right-0 z-50 px-4">
    <div class="max-w-7xl mx-auto glass rounded-full shadow-lg" style="background: rgba(255, 255, 255, 0.6);">
      <div class="px-6">
        <div class="flex justify-between h-16">
          <div class="flex">
            <router-link to="/" class="flex-shrink-0 flex items-center">
              <span class="text-xl font-black tracking-tighter text-black font-heading">BLUEBELL</span>
            </router-link>
            
            <!-- Search Bar -->
            <div class="hidden sm:ml-8 sm:flex sm:items-center">
              <form @submit.prevent="handleSearch" class="relative">
                <input 
                  v-model="searchKeyword"
                  type="text" 
                  class="block w-64 pl-4 pr-10 py-2 border border-black/10 rounded-full leading-5 bg-white/50 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-black/20 sm:text-sm backdrop-blur-sm transition-all focus:w-80" 
                  placeholder="Search Bluebell"
                >
                <button type="submit" class="absolute inset-y-0 right-0 pr-3 flex items-center">
                  <svg class="h-4 w-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path></svg>
                </button>
              </form>
            </div>
          </div>
          <div class="flex items-center space-x-6">
            <template v-if="authStore.token">
              <router-link :to="`/user/${authStore.user?.user_id}`" class="text-sm font-semibold text-black hover:underline">
                Welcome, {{ authStore.user?.username }}
              </router-link>
              <router-link
                v-if="authStore.isAdmin()"
                to="/create-community"
                class="text-sm font-bold text-gray-600 hover:text-black transition-colors"
              >
                New Community
              </router-link>
              <button @click="logout" class="text-sm font-bold text-gray-500 hover:text-black transition-colors cursor-pointer">
                Logout
              </button>
              <router-link to="/create-post" class="bg-black text-white px-6 py-2 rounded-full text-sm font-bold shadow-md hover:bg-gray-800 transition-all active:scale-95">
                New Post
              </router-link>
            </template>
            <template v-else>
              <router-link to="/login" class="text-sm font-bold text-gray-500 hover:text-black transition-colors">
                Login
              </router-link>
              <router-link to="/signup" class="bg-black text-white px-6 py-2 rounded-full text-sm font-bold shadow-md hover:bg-gray-800 transition-all active:scale-95">
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