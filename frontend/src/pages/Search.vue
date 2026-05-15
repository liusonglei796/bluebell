<template>
  <div class="max-w-4xl mx-auto py-6">
    <div class="mb-8">
      <h1 class="text-2xl font-bold text-gray-900 mb-2">
        Search results for: <span class="text-indigo-600">"{{ query }}"</span>
      </h1>
      <p class="text-sm text-gray-500" v-if="!loading">
        Found {{ total }} results
      </p>
    </div>

    <div v-if="loading" class="text-center py-10">
      <span class="text-gray-500">Searching...</span>
    </div>

    <div v-else class="space-y-6">
      <div v-for="post in posts" :key="post.post_id" 
        class="bg-white p-6 border border-gray-200 rounded-lg shadow-sm hover:border-gray-300 transition-colors cursor-pointer"
        @click="goToPost(post.post_id)">
        
        <!-- Highlighted Title -->
        <h3 class="text-lg font-bold text-gray-900 mb-2 group-hover:text-indigo-600 transition-colors" v-if="post.highlight_title" v-html="post.highlight_title[0]"></h3>
        <h3 class="text-lg font-bold text-gray-900 mb-2 group-hover:text-indigo-600 transition-colors" v-else>{{ post.post_title }}</h3>
        
        <div class="text-xs text-gray-500 mb-3 flex items-center gap-2">
          <span class="px-2 py-0.5 bg-gray-100 rounded text-gray-600">Community {{ post.community_id }}</span>
          <span>&bull;</span>
          <span>{{ new Date(post.created_at).toLocaleString() }}</span>
        </div>

        <!-- Highlighted Content -->
        <div class="text-sm text-gray-700 line-clamp-3 prose prose-sm max-w-none" 
          v-if="post.highlight_content" 
          v-html="post.highlight_content.join(' ... ')">
        </div>
        <p class="text-sm text-gray-700 line-clamp-3" v-else>{{ post.content }}</p>
      </div>

      <div v-if="posts.length === 0" class="text-center py-10 bg-white rounded-lg border border-dashed border-gray-300">
        <p class="text-gray-500">No matches found for your search.</p>
        <router-link to="/" class="text-indigo-600 hover:underline mt-2 inline-block">Go back home</router-link>
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
  background-color: #fef08a; /* yellow-200 */
  font-weight: 600;
  padding: 0 2px;
  border-radius: 2px;
}
</style>
