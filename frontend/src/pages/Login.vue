<template>
  <div class="max-w-md mx-auto bg-white p-8 border border-gray-200 rounded-lg shadow-sm mt-10">
    <h2 class="text-2xl font-bold mb-6 text-center text-gray-900">Log in to Bluebell</h2>
    <form @submit.prevent="handleLogin" class="space-y-4">
      <div>
        <label for="username" class="block text-sm font-medium text-gray-700">Username</label>
        <input v-model="form.username" type="text" id="username" required class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm" />
      </div>
      <div>
        <label for="password" class="block text-sm font-medium text-gray-700">Password</label>
        <input v-model="form.password" type="password" id="password" required class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm" />
      </div>
      <div v-if="errorMsg" class="text-red-600 text-sm">
        {{ errorMsg }}
      </div>
      <button type="submit" :disabled="loading" class="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50">
        {{ loading ? 'Logging in...' : 'Login' }}
      </button>
    </form>
    <div class="mt-4 text-center text-sm text-gray-600">
      Don't have an account? <router-link to="/signup" class="font-medium text-indigo-600 hover:text-indigo-500">Sign up</router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '../store/auth';
import request from '../api/request';

const form = ref({ username: '', password: '' });
const loading = ref(false);
const errorMsg = ref('');
const router = useRouter();
const authStore = useAuthStore();

const handleLogin = async () => {
  loading.value = true;
  errorMsg.value = '';
  try {
    const res: any = await request.post('/login', form.value);
    if (res.code === 1000) {
      authStore.setAuth(res.data.access_token, res.data.refresh_token, res.data.username, res.data.user_id, res.data.role);
      router.push('/');
    } else {
      errorMsg.value = res.msg || 'Login failed';
    }
  } catch (err: any) {
    errorMsg.value = err.response?.data?.msg || 'An error occurred';
  } finally {
    loading.value = false;
  }
};
</script>