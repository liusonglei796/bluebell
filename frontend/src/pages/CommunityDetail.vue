<template>
  <div class="flex flex-col md:flex-row gap-6">
    <div class="flex-grow space-y-4 md:w-2/3">
      <div class="flex justify-between items-center mb-6">
        <h1 class="text-3xl font-black text-black tracking-tighter uppercase font-heading">{{ community?.name || 'Loading Community...' }}</h1>
        <div class="flex p-1 bg-white/50 backdrop-blur-md rounded-xl border border-black/5">
          <button @click="changeOrder('score')" :class="['px-4 py-1.5 text-sm font-bold rounded-lg transition-all duration-200', order === 'score' ? 'bg-black text-white shadow-sm' : 'text-gray-500 hover:text-black']">Hot</button>
          <button @click="changeOrder('time')" :class="['px-4 py-1.5 text-sm font-bold rounded-lg transition-all duration-200', order === 'time' ? 'bg-black text-white shadow-sm' : 'text-gray-500 hover:text-black']">New</button>
        </div>
      </div>
      
      <div v-if="community" class="glass rounded-2xl p-5 border-none mb-6">
        <p class="text-gray-700 font-medium leading-relaxed">{{ community.introduction }}</p>
      </div>

      <div v-if="loading" class="flex flex-col items-center justify-center py-20 space-y-4">
        <div class="w-12 h-12 border-4 border-gray-200 border-t-black rounded-full animate-spin"></div>
        <span class="text-black/60 font-bold uppercase tracking-widest text-xs">Loading posts...</span>
      </div>
      
      <template v-else>
        <PostCard v-for="post in posts" :key="post.id" :post="post" @voted="fetchPosts" />
        <div v-if="posts.length === 0" class="glass rounded-[24px] p-10 text-center">
          <span class="text-gray-400 font-medium italic">No posts in this community yet.</span>
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