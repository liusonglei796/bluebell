<template>
  <div class="max-w-2xl mx-auto bg-white p-8 border border-gray-200 rounded-lg shadow-sm">
    <h2 class="text-2xl font-bold mb-6 text-gray-900">Create a New Post</h2>
    <form @submit.prevent="submitPost" class="space-y-6">
      <div>
        <label for="community" class="block text-sm font-medium text-gray-700">Community</label>
        <select v-model="form.community_id" id="community" required class="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md shadow-sm border">
          <option disabled value="">Select a community</option>
          <option v-for="community in communities" :key="community.id" :value="community.id">
            {{ community.name }}
          </option>
        </select>
      </div>
      <div>
        <label for="title" class="block text-sm font-medium text-gray-700">Title</label>
        <input v-model="form.title" type="text" id="title" required class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm" />
      </div>
      <div>
        <label for="content" class="block text-sm font-medium text-gray-700">Content</label>
        <textarea v-model="form.content" id="content" rows="6" required class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"></textarea>
      </div>
      <div v-if="errorMsg" class="text-red-600 text-sm">
        {{ errorMsg }}
      </div>
      <div class="flex justify-end">
        <button type="submit" :disabled="loading" class="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50">
          {{ loading ? 'Posting...' : 'Post' }}
        </button>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import request from '../api/request';

const communities = ref<any[]>([]);
const form = ref({
  community_id: '',
  title: '',
  content: '',
});
const loading = ref(false);
const errorMsg = ref('');
const router = useRouter();

const fetchCommunities = async () => {
  try {
    const res: any = await request.get('/community');
    if (res.code === 1000) {
      communities.value = res.data || [];
    }
  } catch (err) {
    console.error('Failed to fetch communities', err);
  }
};

const submitPost = async () => {
  loading.value = true;
  errorMsg.value = '';
  try {
    // API expects community_id as integer
    const payload = {
      ...form.value,
      community_id: parseInt(form.value.community_id as string, 10),
    };
    const res: any = await request.post('/post', payload);
    if (res.code === 1000) {
      router.push('/');
    } else {
      errorMsg.value = res.msg || 'Failed to create post';
    }
  } catch (err: any) {
    errorMsg.value = err.response?.data?.msg || 'An error occurred';
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  fetchCommunities();
});
</script>