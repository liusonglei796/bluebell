<template>
  <div class="max-w-4xl mx-auto">
    <div v-if="loading" class="text-center py-10">
      <span class="text-gray-500">Loading post...</span>
    </div>

    <div v-else-if="post" class="bg-white p-6 border border-gray-200 rounded-lg shadow-sm">
      <div class="flex">
        <div class="flex flex-col items-center mr-6">
          <button @click="vote(1)" class="text-gray-400 hover:text-indigo-600 focus:outline-none">
            <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7"></path></svg>
          </button>
          <span class="text-xl font-bold text-gray-700 my-2">{{ post.score || 0 }}</span>
          <button @click="vote(-1)" class="text-gray-400 hover:text-red-600 focus:outline-none">
            <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
          </button>
        </div>
        <div class="flex-grow">
          <div class="text-sm text-gray-500 mb-2">
            Posted by <span class="font-medium text-gray-900">{{ post.author_name }}</span> 
            in <router-link :to="`/community/${post.community_id}`" class="font-medium text-indigo-600 hover:underline">{{ post.community?.name || `Community ${post.community_id}` }}</router-link>
          </div>
          <h1 class="text-2xl font-bold text-gray-900 mb-4">{{ post.title }}</h1>
          <div class="prose max-w-none text-gray-800 whitespace-pre-wrap">
            {{ post.content }}
          </div>
        </div>
      </div>
    </div>

    <div v-else class="text-center py-10">
      <span class="text-gray-500">Post not found.</span>
    </div>

    <!-- Remarks Section -->
    <div v-if="post" class="mt-8">
      <h2 class="text-xl font-bold text-gray-900 mb-4">Comments</h2>
      
      <!-- Add Comment -->
      <form @submit.prevent="submitRemark" class="mb-8">
        <textarea v-model="newRemark" rows="3" class="w-full p-3 border border-gray-300 rounded-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500" placeholder="What are your thoughts?"></textarea>
        <div class="mt-2 flex justify-end">
          <button type="submit" :disabled="submittingRemark || !newRemark.trim()" class="px-4 py-2 bg-indigo-600 text-white rounded-md text-sm font-medium hover:bg-indigo-700 disabled:opacity-50">
            {{ submittingRemark ? 'Posting...' : 'Comment' }}
          </button>
        </div>
      </form>

      <!-- Comment List -->
      <div class="space-y-4">
        <div v-for="remark in remarks" :key="remark.id" class="bg-white p-4 border border-gray-200 rounded-lg shadow-sm">
          <div class="text-sm text-gray-500 mb-2">
            <span class="font-medium text-gray-900">{{ remark.author_name || 'User ' + remark.author_id }}</span>
            &bull; {{ new Date(remark.create_time).toLocaleString() }}
          </div>
          <p class="text-gray-800 whitespace-pre-wrap">{{ remark.content }}</p>
        </div>
        <div v-if="remarks.length === 0" class="text-gray-500 text-sm">
          No comments yet. Be the first to share your thoughts!
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import request from '../api/request';

const route = useRoute();
const router = useRouter();
const postId = ref(Number(route.params.id));
const post = ref<any>(null);
const remarks = ref<any[]>([]);
const loading = ref(true);
const newRemark = ref('');
const submittingRemark = ref(false);

const fetchPost = async () => {
  try {
    const res: any = await request.get(`/post/${postId.value}`);
    if (res.code === 1000) {
      post.value = res.data;
    }
  } catch (err) {
    console.error('Failed to fetch post', err);
  } finally {
    loading.value = false;
  }
};

const fetchRemarks = async () => {
  try {
    const res: any = await request.get(`/post/${postId.value}/remarks`);
    if (res.code === 1000) {
      remarks.value = res.data || [];
    }
  } catch (err) {
    console.error('Failed to fetch remarks', err);
  }
};

const vote = async (direction: number) => {
  try {
    const res: any = await request.post('/vote', {
      post_id: postId.value,
      direction,
    });
    if (res.code === 1000) {
      alert('Vote successful!');
      fetchPost(); // Refresh post score
    } else {
      alert(res.msg || 'Vote failed');
    }
  } catch (err: any) {
    if (err.response?.status === 401) {
      router.push('/login');
    } else {
      alert('Vote failed');
    }
  }
};

const submitRemark = async () => {
  if (!newRemark.value.trim()) return;
  submittingRemark.value = true;
  try {
    const res: any = await request.post('/remark', {
      post_id: postId.value,
      content: newRemark.value,
    });
    if (res.code === 1000) {
      newRemark.value = '';
      fetchRemarks(); // Refresh comments
    } else {
      alert(res.msg || 'Failed to post comment');
    }
  } catch (err: any) {
    if (err.response?.status === 401) {
      router.push('/login');
    } else {
      alert('Failed to post comment');
    }
  } finally {
    submittingRemark.value = false;
  }
};

onMounted(() => {
  fetchPost();
  fetchRemarks();
});
</script>