<template>
  <div class="max-w-lg mx-auto bg-white p-8 border border-gray-200 rounded-lg shadow-sm mt-10">
    <h2 class="text-2xl font-bold mb-6 text-center text-gray-900">Create New Community</h2>
    <form @submit.prevent="handleCreate" class="space-y-4">
      <div>
        <label for="name" class="block text-sm font-medium text-gray-700">Community Name</label>
        <input
          v-model="form.name"
          type="text"
          id="name"
          required
          placeholder="e.g., Go, Python, React"
          class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
        />
      </div>
      <div>
        <label for="introduction" class="block text-sm font-medium text-gray-700">Introduction</label>
        <textarea
          v-model="form.introduction"
          id="introduction"
          required
          rows="4"
          placeholder="Brief description of this community..."
          class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
        ></textarea>
      </div>
      <div v-if="errorMsg" class="text-red-600 text-sm">
        {{ errorMsg }}
      </div>
      <div v-if="successMsg" class="text-green-600 text-sm">
        {{ successMsg }}
      </div>
      <button
        type="submit"
        :disabled="loading"
        class="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
      >
        {{ loading ? 'Creating...' : 'Create Community' }}
      </button>
    </form>
    <div class="mt-4 text-center text-sm text-gray-600">
      <router-link to="/" class="font-medium text-indigo-600 hover:text-indigo-500">Back to Home</router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '../store/auth';
import request from '../api/request';

const form = ref({ name: '', introduction: '' });
const loading = ref(false);
const errorMsg = ref('');
const successMsg = ref('');
const router = useRouter();
const authStore = useAuthStore();

// 检查权限
if (!authStore.isAdmin()) {
  errorMsg.value = 'You do not have permission to create a community.';
}

const handleCreate = async () => {
  if (!authStore.isAdmin()) {
    errorMsg.value = 'You do not have permission to create a community.';
    return;
  }

  loading.value = true;
  errorMsg.value = '';
  successMsg.value = '';

  try {
    const res: any = await request.post('/community', form.value);
    if (res.code === 1000) {
      successMsg.value = 'Community created successfully!';
      form.value = { name: '', introduction: '' };
      // 2秒后跳转到社区列表
      setTimeout(() => {
        router.push('/');
      }, 1500);
    } else {
      errorMsg.value = res.msg || 'Failed to create community';
    }
  } catch (err: any) {
    errorMsg.value = err.response?.data?.msg || 'An error occurred';
  } finally {
    loading.value = false;
  }
};
</script>
