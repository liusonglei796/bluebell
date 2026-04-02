<template>
  <div class="flex flex-col md:flex-row gap-6">
    <div class="flex-grow space-y-4 md:w-2/3">
      <div class="flex justify-between items-center mb-6">
        <h1 class="text-2xl font-bold text-gray-900">Popular Posts</h1>
        <div class="flex gap-2">
          <button @click="changeOrder('score')" :class="['px-3 py-1 text-sm rounded-md', order === 'score' ? 'bg-indigo-600 text-white' : 'bg-white text-gray-700 border hover:bg-gray-50']">Hot</button>
          <button @click="changeOrder('time')" :class="['px-3 py-1 text-sm rounded-md', order === 'time' ? 'bg-indigo-600 text-white' : 'bg-white text-gray-700 border hover:bg-gray-50']">New</button>
        </div>
      </div>
      
      <div v-if="loading" class="text-center py-10">
        <span class="text-gray-500">Loading posts...</span>
      </div>
      
      <template v-else>
        <PostCard v-for="post in posts" :key="post.id" :post="post" />
        <div v-if="posts.length === 0" class="text-center py-10">
          <span class="text-gray-500">No posts found.</span>
        </div>
      </template>
    </div>
    
    <div class="md:w-1/3">
      <div class="bg-white p-4 border border-gray-200 rounded-lg shadow-sm">
        <h2 class="text-lg font-bold text-gray-900 mb-4">Communities</h2>
        <ul class="space-y-2">
          <li v-for="community in communities" :key="community.id">
            <router-link :to="`/community/${community.id}`" class="block px-3 py-2 rounded-md hover:bg-gray-50 text-gray-700">
              <span class="font-medium">{{ community.name }}</span>
            </router-link>
          </li>
        </ul>
        <div v-if="loadingCommunities" class="text-gray-500 text-sm py-2 italic text-center">Loading communities...</div>
        <div v-else-if="communities.length === 0" class="text-gray-400 text-sm py-2 text-center">No communities found</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import request from '../api/request';
import PostCard from '../components/PostCard.vue';

const posts = ref<any[]>([]);
const communities = ref<any[]>([]);
const loading = ref(true);
const loadingCommunities = ref(true);
const order = ref('score');

const fetchPosts = async () => {
  loading.value = true;
  try {
    const res: any = await request.get(`/posts?page=1&size=20&order=${order.value}`);
    if (res.code === 1000) {
      posts.value = res.data || [];
    }
  } catch (err) {
    console.error('Failed to fetch posts', err);
  } finally {
    loading.value = false;
  }
};

const fetchCommunities = async () => {
  loadingCommunities.value = true;
  try {
    const res: any = await request.get('/community');
    if (res.code === 1000) {
      communities.value = res.data || [];
    }
  } catch (err) {
    console.error('Failed to fetch communities', err);
  } finally {
    loadingCommunities.value = false;
  }
};

const changeOrder = (newOrder: string) => {
  order.value = newOrder;
  fetchPosts();
};

onMounted(() => {
  fetchPosts();
  fetchCommunities();
});
</script>