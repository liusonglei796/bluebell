<template>
  <div class="flex flex-col md:flex-row gap-8">
    <div class="flex-grow space-y-6 md:w-2/3">
      <div class="flex justify-between items-center mb-4">
        <h1 class="text-3xl font-black text-black tracking-tighter">POPULAR POSTS</h1>
        <div class="flex p-1 bg-white/50 backdrop-blur-md rounded-xl border border-black/5">
          <button @click="changeOrder('score')" 
                  :class="['px-4 py-1.5 text-sm font-bold rounded-lg transition-all duration-200', 
                           order === 'score' ? 'bg-black text-white shadow-sm' : 'text-gray-500 hover:text-black']">
            HOT
          </button>
          <button @click="changeOrder('time')" 
                  :class="['px-4 py-1.5 text-sm font-bold rounded-lg transition-all duration-200', 
                           order === 'time' ? 'bg-black text-white shadow-sm' : 'text-gray-500 hover:text-black']">
            NEW
          </button>
        </div>
      </div>
      
      <div v-if="loading" class="flex flex-col items-center justify-center py-20 space-y-4">
        <div class="w-12 h-12 border-4 border-gray-200 border-t-black rounded-full animate-spin"></div>
        <span class="text-black/60 font-bold uppercase tracking-widest text-xs">Loading posts...</span>
      </div>
      
      <template v-else>
        <PostCard v-for="post in posts" :key="post.id" :post="post" />
        <div v-if="posts.length === 0" class="glass rounded-[24px] p-10 text-center">
          <span class="text-gray-400 font-medium">No posts found in this universe.</span>
        </div>
      </template>
    </div>
    
    <div class="md:w-1/3">
      <div class="glass rounded-[24px] p-6 sticky top-24 shadow-sm">
        <h2 class="text-xl font-black text-black mb-6 flex items-center gap-2 tracking-tight">
          <UsersIcon class="w-5 h-5 text-black" />
          COMMUNITIES
        </h2>
        <ul class="space-y-2">
          <li v-for="community in communities" :key="community.id">
            <router-link :to="`/community/${community.id}`" 
                         class="flex items-center gap-3 px-4 py-3 rounded-2xl hover:bg-black/5 transition-all duration-200 text-gray-600 hover:text-black font-bold group border border-transparent">
              <div class="w-8 h-8 rounded-full bg-black/5 flex items-center justify-center text-xs text-black font-black group-hover:scale-110 transition-transform">
                {{ community.name.substring(0, 1).toUpperCase() }}
              </div>
              {{ community.name }}
            </router-link>
          </li>
        </ul>
        <div v-if="loadingCommunities" class="flex items-center justify-center py-6">
          <div class="w-6 h-6 border-2 border-gray-100 border-t-black rounded-full animate-spin"></div>
        </div>
        <div v-else-if="communities.length === 0" class="text-gray-400 text-sm py-6 text-center italic">No communities found</div>
        
        <button class="w-full mt-6 py-4 px-4 rounded-2xl bg-black text-white font-black hover:bg-gray-800 transition-all duration-300 shadow-xl shadow-black/10 active:scale-95 uppercase tracking-wider text-sm">
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