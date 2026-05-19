<template>
  <div class="flex flex-col md:flex-row gap-8">
    <div class="flex-grow space-y-6 md:w-2/3">
      <div class="flex justify-between items-center mb-4">
        <h1 class="text-3xl font-extrabold text-[#1e1b4b] tracking-tight">Popular Posts</h1>
        <div class="flex p-1 bg-white/30 backdrop-blur-md rounded-xl border border-white/20">
          <button @click="changeOrder('score')" 
                  :class="['px-4 py-1.5 text-sm font-semibold rounded-lg transition-all duration-200', 
                           order === 'score' ? 'bg-white text-indigo-700 shadow-sm' : 'text-gray-600 hover:text-indigo-600']">
            Hot
          </button>
          <button @click="changeOrder('time')" 
                  :class="['px-4 py-1.5 text-sm font-semibold rounded-lg transition-all duration-200', 
                           order === 'time' ? 'bg-white text-indigo-700 shadow-sm' : 'text-gray-600 hover:text-indigo-600']">
            New
          </button>
        </div>
      </div>
      
      <div v-if="loading" class="flex flex-col items-center justify-center py-20 space-y-4">
        <div class="w-12 h-12 border-4 border-indigo-200 border-t-indigo-600 rounded-full animate-spin"></div>
        <span class="text-indigo-900/60 font-medium">Loading posts...</span>
      </div>
      
      <template v-else>
        <PostCard v-for="post in posts" :key="post.id" :post="post" />
        <div v-if="posts.length === 0" class="glass rounded-[24px] p-10 text-center">
          <span class="text-gray-500 text-lg">No posts found in this universe.</span>
        </div>
      </template>
    </div>
    
    <div class="md:w-1/3">
      <div class="glass rounded-[24px] p-6 sticky top-24 shadow-sm">
        <h2 class="text-xl font-bold text-[#1e1b4b] mb-6 flex items-center gap-2">
          <UsersIcon class="w-5 h-5 text-indigo-600" />
          Communities
        </h2>
        <ul class="space-y-2">
          <li v-for="community in communities" :key="community.id">
            <router-link :to="`/community/${community.id}`" 
                         class="flex items-center gap-3 px-4 py-3 rounded-2xl hover:bg-white/40 transition-all duration-200 text-gray-700 hover:text-indigo-700 font-medium group border border-transparent hover:border-white/40">
              <div class="w-8 h-8 rounded-full bg-gradient-to-br from-indigo-100 to-purple-100 flex items-center justify-center text-xs text-indigo-600 font-bold group-hover:scale-110 transition-transform">
                {{ community.name.substring(0, 1).toUpperCase() }}
              </div>
              {{ community.name }}
            </router-link>
          </li>
        </ul>
        <div v-if="loadingCommunities" class="flex items-center justify-center py-6">
          <div class="w-6 h-6 border-2 border-indigo-100 border-t-indigo-500 rounded-full animate-spin"></div>
        </div>
        <div v-else-if="communities.length === 0" class="text-gray-400 text-sm py-6 text-center italic">No communities found</div>
        
        <button class="w-full mt-6 py-3 px-4 rounded-2xl bg-indigo-600 text-white font-bold hover:bg-indigo-700 transition-all duration-300 shadow-lg shadow-indigo-200 hover:shadow-indigo-300">
          Explore All
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { UsersIcon } from 'lucide-vue-next';
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