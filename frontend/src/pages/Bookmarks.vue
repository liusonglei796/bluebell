<template>
  <div class="max-w-4xl mx-auto px-4 pt-24 pb-12">
    <h1 class="text-3xl font-black tracking-tighter mb-8 font-heading">My Bookmarks</h1>

    <div v-if="loading" class="flex flex-col items-center justify-center py-20 space-y-4">
      <div class="w-12 h-12 border-4 border-gray-200 border-t-black rounded-full animate-spin"></div>
      <span class="text-black/60 font-bold uppercase tracking-widest text-xs">Loading...</span>
    </div>

    <div v-else-if="bookmarks.length > 0" class="space-y-6">
      <PostCard 
        v-for="post in bookmarks" 
        :key="post.id" 
        :post="post"
        @voted="fetchBookmarks"
      />
    </div>

    <div v-else class="glass rounded-3xl p-12 text-center">
      <div class="mb-4">
        <BookmarkIcon :size="48" class="mx-auto text-gray-300" />
      </div>
      <p class="text-gray-500 font-bold italic text-lg">You haven't bookmarked any posts yet.</p>
      <p class="text-gray-400 text-sm mt-2">Start bookmarking posts you want to save for later!</p>
      <router-link to="/" class="mt-6 inline-block bg-black text-white px-6 py-3 rounded-2xl font-bold hover:bg-gray-800 transition-all">
        Explore Posts
      </router-link>
    </div>
  </div>
</template>

<script setup lang="ts">import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '../store/auth';
import request from '../api/request';
import PostCard from '../components/PostCard.vue';
import { BookmarkIcon } from 'lucide-vue-next';
const router = useRouter();
const authStore = useAuthStore();
const bookmarks = ref<any[]>([]);
const loading = ref(true);
const fetchBookmarks = async () => {
  if (!authStore.user?.user_id) {
    router.push('/login');
    return;
  }
  loading.value = true;
  try {
    const res: any = await request.get(`/user/${authStore.user.user_id}/bookmarks?page=1&page_size=20`);
    if (res.code === 200) {
      bookmarks.value = res.data.list || [];
    }
  } catch (err: any) {
    if (err.response?.status === 401) {
      router.push('/login');
    } else {
      console.error('Failed to fetch bookmarks', err);
    }
  } finally {
    loading.value = false;
  }
};
onMounted(() => {
  if (!authStore.token) {
    router.push('/login');
  } else {
    fetchBookmarks();
  }
});</script>
