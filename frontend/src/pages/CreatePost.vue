<template>
  <div class="flex items-center justify-center min-h-[60vh]">
    <div class="glass rounded-3xl p-10 w-full max-w-2xl">
      <h2 class="text-3xl font-black mb-8 text-black tracking-tighter font-heading">NEW POST</h2>
      <form @submit.prevent="submitPost" class="space-y-6">
        <div>
          <label for="community" class="block text-xs font-bold uppercase tracking-widest text-gray-500 mb-2">Community</label>
          <select v-model="form.community_id" id="community" required class="block w-full px-4 py-3 bg-white/50 border border-black/10 rounded-xl focus:outline-none focus:ring-2 focus:ring-black/20 backdrop-blur-sm text-sm appearance-none">
            <option disabled value="">Select a community</option>
            <option v-for="community in communities" :key="community.id" :value="community.id">
              {{ community.name }}
            </option>
          </select>
        </div>
        <div>
          <label for="title" class="block text-xs font-bold uppercase tracking-widest text-gray-500 mb-2">Title</label>
          <input v-model="form.title" type="text" id="title" required class="block w-full px-4 py-3 bg-white/50 border border-black/10 rounded-xl focus:outline-none focus:ring-2 focus:ring-black/20 backdrop-blur-sm text-sm" />
        </div>
        <div>
          <label for="content" class="block text-xs font-bold uppercase tracking-widest text-gray-500 mb-2">Content</label>
          <textarea v-model="form.content" id="content" rows="6" required class="block w-full px-4 py-3 bg-white/50 border border-black/10 rounded-xl focus:outline-none focus:ring-2 focus:ring-black/20 backdrop-blur-sm text-sm resize-none"></textarea>
        </div>
        <div v-if="errorMsg" class="text-red-500 font-medium text-sm">
          {{ errorMsg }}
        </div>
        <div class="flex justify-end">
          <button type="submit" :disabled="loading" class="px-8 py-3 rounded-2xl bg-black text-white font-black text-sm uppercase tracking-wider hover:bg-gray-800 transition-all active:scale-95 shadow-lg disabled:opacity-50">
            {{ loading ? 'Posting...' : 'Publish' }}
          </button>
        </div>
      </form>
    </div>
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