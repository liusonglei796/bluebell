<template>
  <div class="max-w-4xl mx-auto py-6">
    <div class="mb-8">
      <h1 class="text-3xl font-black text-black tracking-tighter mb-2 font-heading">
        SEARCH: <span class="text-gray-400">"{{ query }}"</span>
      </h1>
      <p class="text-sm text-gray-500 font-bold uppercase tracking-widest" v-if="!loading">
        {{ total }} results found
      </p>
    </div>

    <div v-if="loading" class="flex flex-col items-center justify-center py-20 space-y-4">
      <div class="w-12 h-12 border-4 border-gray-200 border-t-black rounded-full animate-spin"></div>
      <span class="text-black/60 font-bold uppercase tracking-widest text-xs">Searching...</span>
    </div>

    <div v-else class="space-y-4">
      <div v-for="post in posts" :key="post.post_id" 
        class="glass rounded-[24px] p-6 hover:-translate-y-1 hover:shadow-md transition-all duration-300 cursor-pointer"
        @click="goToPost(post.post_id)">
        
        <h3 class="text-lg font-black text-black mb-2 tracking-tight font-heading" v-if="post.highlight_title" v-html="post.highlight_title[0]"></h3>
        <h3 class="text-lg font-black text-black mb-2 tracking-tight font-heading" v-else>{{ post.post_title }}</h3>
        
        <div class="text-xs text-gray-500 mb-3 flex items-center gap-2">
          <span class="bg-black/5 px-2 py-0.5 rounded-full text-black font-bold border border-black/5">Community {{ post.community_id }}</span>
          <span>•</span>
          <span>{{ new Date(post.created_at).toLocaleString() }}</span>
        </div>

        <div class="text-sm text-gray-600 line-clamp-3 leading-relaxed" 
          v-if="post.highlight_content" 
          v-html="post.highlight_content.join(' ... ')">
        </div>
        <p class="text-sm text-gray-600 line-clamp-3 leading-relaxed" v-else>{{ post.content }}</p>
      </div>

      <div v-if="posts.length === 0" class="glass rounded-[24px] p-12 text-center">
        <p class="text-gray-400 font-medium italic mb-4">No matches found for your search.</p>
        <router-link to="/" class="text-xs font-black uppercase tracking-widest text-black hover:underline">← Go back home</router-link>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import request from '../api/request';

const route = useRoute();
const router = useRouter();
const query = ref(route.query.q as string || '');
const posts = ref<any[]>([]);
const total = ref(0);
const loading = ref(false);

const performSearch = async () => {
  if (!query.value) return;
  
  loading.value = true;
  try {
    const res: any = await request.get(`/search?keyword=${encodeURIComponent(query.value)}&page=1&page_size=20`);
    if (res.code === 1000) {
      posts.value = res.data.posts || [];
      total.value = res.data.total;
    }
  } catch (err) {
    console.error('Search failed', err);
  } finally {
    loading.value = false;
  }
};

const goToPost = (id: string) => {
  router.push(`/post/${id}`);
};

// Watch for query changes (e.g., when searching again from the navbar)
watch(() => route.query.q, (newQ) => {
  query.value = newQ as string || '';
  performSearch();
});

onMounted(() => {
  performSearch();
});
</script>

<style>
.highlight {
  background-color: rgba(0, 0, 0, 0.08);
  font-weight: 700;
  padding: 1px 4px;
  border-radius: 4px;
}
</style>
