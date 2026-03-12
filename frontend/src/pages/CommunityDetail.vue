<template>
  <div class="flex flex-col md:flex-row gap-6">
    <div class="flex-grow space-y-4 md:w-2/3">
      <div class="flex justify-between items-center mb-6">
        <h1 class="text-2xl font-bold text-gray-900">{{ community?.name || 'Loading Community...' }}</h1>
        <div class="flex gap-2">
          <button @click="changeOrder('score')" :class="['px-3 py-1 text-sm rounded-md', order === 'score' ? 'bg-indigo-600 text-white' : 'bg-white text-gray-700 border hover:bg-gray-50']">Hot</button>
          <button @click="changeOrder('time')" :class="['px-3 py-1 text-sm rounded-md', order === 'time' ? 'bg-indigo-600 text-white' : 'bg-white text-gray-700 border hover:bg-gray-50']">New</button>
        </div>
      </div>
      
      <div v-if="community" class="bg-indigo-50 p-4 rounded-lg border border-indigo-100 mb-6">
        <p class="text-indigo-800">{{ community.introduction }}</p>
      </div>

      <div v-if="loading" class="text-center py-10">
        <span class="text-gray-500">Loading posts...</span>
      </div>
      
      <template v-else>
        <PostCard v-for="post in posts" :key="post.id" :post="post" />
        <div v-if="posts.length === 0" class="text-center py-10">
          <span class="text-gray-500">No posts in this community yet.</span>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue';
import { useRoute } from 'vue-router';
import request from '../api/request';
import PostCard from '../components/PostCard.vue';

const route = useRoute();
const communityId = ref(route.params.id);
const community = ref<any>(null);
const posts = ref<any[]>([]);
const loading = ref(true);
const order = ref('score');

const fetchCommunityDetail = async () => {
  try {
    const res: any = await request.get(`/community/${communityId.value}`);
    if (res.code === 1000) {
      community.value = res.data;
    }
  } catch (err) {
    console.error('Failed to fetch community details', err);
  }
};

const fetchPosts = async () => {
  loading.value = true;
  try {
    const res: any = await request.get(`/posts?page=1&size=20&order=${order.value}&community_id=${communityId.value}`);
    if (res.code === 1000) {
      posts.value = res.data || [];
    }
  } catch (err) {
    console.error('Failed to fetch posts', err);
  } finally {
    loading.value = false;
  }
};

const changeOrder = (newOrder: string) => {
  order.value = newOrder;
  fetchPosts();
};

onMounted(() => {
  fetchCommunityDetail();
  fetchPosts();
});

watch(() => route.params.id, (newId) => {
  if (newId) {
    communityId.value = newId;
    fetchCommunityDetail();
    fetchPosts();
  }
});
</script>