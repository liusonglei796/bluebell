<template>
  <div class="flex items-center justify-center min-h-[60vh]">
    <div class="glass rounded-3xl p-10 w-full max-w-lg">
      <h2 class="text-3xl font-black text-center text-black tracking-tighter mb-8 font-heading">NEW COMMUNITY</h2>
      <form @submit.prevent="handleCreate" class="space-y-5">
        <div>
          <label for="name" class="block text-xs font-bold uppercase tracking-widest text-gray-500 mb-2">Community Name</label>
          <input
            v-model="form.name"
            type="text"
            id="name"
            required
            placeholder="e.g., Go, Python, React"
            class="block w-full px-4 py-3 bg-white/50 border border-black/10 rounded-xl focus:outline-none focus:ring-2 focus:ring-black/20 backdrop-blur-sm text-sm"
          />
        </div>
        <div>
          <label for="introduction" class="block text-xs font-bold uppercase tracking-widest text-gray-500 mb-2">Introduction</label>
          <textarea
            v-model="form.introduction"
            id="introduction"
            required
            rows="4"
            placeholder="Brief description of this community..."
            class="block w-full px-4 py-3 bg-white/50 border border-black/10 rounded-xl focus:outline-none focus:ring-2 focus:ring-black/20 backdrop-blur-sm text-sm resize-none"
          ></textarea>
        </div>
        <div v-if="errorMsg" class="text-red-500 font-medium text-sm">
          {{ errorMsg }}
        </div>
        <div v-if="successMsg" class="text-green-600 font-medium text-sm">
          {{ successMsg }}
        </div>
        <button
          type="submit"
          :disabled="loading"
          class="w-full py-4 rounded-2xl bg-black text-white font-black text-sm uppercase tracking-wider hover:bg-gray-800 transition-all active:scale-95 shadow-lg disabled:opacity-50"
        >
          {{ loading ? 'Creating...' : 'Create Community' }}
        </button>
      </form>
      <div class="mt-6 text-center text-sm text-gray-500">
        <router-link to="/" class="font-bold text-black hover:underline">← Back to Home</router-link>
      </div>
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
