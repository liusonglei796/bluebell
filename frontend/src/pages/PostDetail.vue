<template>
  <div class="max-w-4xl mx-auto">
    <div v-if="loading" class="flex flex-col items-center justify-center py-20 space-y-4">
      <div class="w-12 h-12 border-4 border-gray-200 border-t-black rounded-full animate-spin"></div>
      <span class="text-black/60 font-bold uppercase tracking-widest text-xs">Loading...</span>
    </div>

    <div v-else-if="post" class="glass rounded-3xl p-8">
      <div class="flex">
        <div class="flex flex-col items-center mr-6">
          <div class="bg-black/5 rounded-2xl p-3 flex flex-col items-center">
            <button @click="vote(1)" class="text-gray-400 hover:text-black transition-colors cursor-pointer focus:outline-none">
              <ArrowBigUpIcon :stroke-width="2.5" class="w-7 h-7" />
            </button>
            <span class="text-xl font-black text-black font-heading my-2">{{ post.vote_num || 0 }}</span>
            <button @click="vote(-1)" class="text-gray-400 hover:text-black transition-colors cursor-pointer focus:outline-none">
              <ArrowBigDownIcon :stroke-width="2.5" class="w-7 h-7" />
            </button>
          </div>
          <div class="mt-4 pt-4 border-t border-black/10">
            <button @click="toggleBookmark" class="text-gray-400 hover:text-black transition-colors cursor-pointer focus:outline-none" :title="isBookmarked ? 'Remove bookmark' : 'Add bookmark'">
              <BookmarkIcon :size="28" :fill="isBookmarked ? 'currentColor' : 'none'" />
            </button>
            <span v-if="bookmarkCount > 0" class="text-xs text-gray-500 mt-1 block">{{ bookmarkCount }}</span>
          </div>
        </div>
        <div class="flex-grow">
          <div class="text-sm text-gray-500 mb-2">
            Posted by <span class="font-bold text-black">{{ post.author_name }}</span>
            in <router-link :to="`/community/${post.community_id}`" class="font-bold text-black hover:underline">{{ post.community?.name || `Community ${post.community_id}` }}</router-link>
          </div>
          <h1 class="text-3xl font-black text-black tracking-tight font-heading mb-4">{{ post.title }}</h1>
          <div class="prose max-w-none text-gray-800 whitespace-pre-wrap">
            {{ post.content }}
          </div>
        </div>
      </div>
    </div>

    <div v-else class="glass rounded-[24px] p-10 text-center">
      <span class="text-gray-400 font-medium italic">Post not found.</span>
    </div>

    <!-- Remarks Section -->
    <div v-if="post" class="mt-8">
      <h2 class="text-2xl font-black text-black tracking-tighter uppercase font-heading mb-4">Comments</h2>

      <!-- Add Comment -->
      <form @submit.prevent="submitRemark" class="mb-8">
        <textarea v-model="newRemark" rows="3" class="w-full p-3 bg-white/50 border border-black/10 rounded-xl focus:outline-none focus:ring-2 focus:ring-black/20 backdrop-blur-sm" placeholder="What are your thoughts?"></textarea>
        <div class="mt-2 flex justify-end">
          <button type="submit" :disabled="submittingRemark || !newRemark.trim()" class="bg-black text-white px-6 py-2.5 rounded-2xl font-bold hover:bg-gray-800 transition-all active:scale-95 shadow-md text-sm disabled:opacity-50">
            {{ submittingRemark ? 'Posting...' : 'Comment' }}
          </button>
        </div>
      </form>

      <!-- Comment List -->
      <div class="space-y-4">
        <div v-for="remark in remarks" :key="remark.id" class="glass rounded-2xl p-5">
          <div class="text-sm text-gray-500 mb-2">
            <span class="font-bold text-black">{{ remark.author_name || 'User ' + remark.author_id }}</span>
            &bull; {{ new Date(remark.create_time).toLocaleString() }}
          </div>
          <p class="text-gray-800 whitespace-pre-wrap">{{ remark.content }}</p>
        </div>
        <div v-if="remarks.length === 0" class="glass rounded-[24px] p-10 text-center">
          <span class="text-gray-400 font-medium italic">No comments yet. Be the first to share your thoughts!</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import request from '../api/request';
import { bookmarkAPI } from '../api/bookmark';
import { useAuthStore } from '../store/auth';
import { ArrowBigUpIcon, ArrowBigDownIcon, BookmarkIcon } from 'lucide-vue-next';

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const postId = ref(route.params.id as string);
const post = ref<any>(null);
const remarks = ref<any[]>([]);
const loading = ref(true);
const newRemark = ref('');
const submittingRemark = ref(false);
const isBookmarked = ref(false);
const bookmarkCount = ref(0);

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

const fetchBookmarkStatus = async () => {
  if (!authStore.token) return;
  try {
    const res: any = await bookmarkAPI.getBookmarkStatus(postId.value);
    if (res.code === 1000) {
      isBookmarked.value = res.data.bookmarked;
      bookmarkCount.value = res.data.count;
    }
  } catch (err) {
    console.error('Failed to fetch bookmark status', err);
  }
};

const toggleBookmark = async () => {
  if (!authStore.token) {
    router.push('/login');
    return;
  }
  try {
    if (isBookmarked.value) {
      await bookmarkAPI.deleteBookmark(postId.value);
      isBookmarked.value = false;
      bookmarkCount.value--;
    } else {
      await bookmarkAPI.createBookmark(postId.value);
      isBookmarked.value = true;
      bookmarkCount.value++;
    }
  } catch (err: any) {
    if (err.response?.status === 401) {
      router.push('/login');
    } else {
      console.error('Bookmark operation failed', err);
    }
  }
};

const vote = async (direction: number) => {
  try {
    const res: any = await request.post('/vote', {
      post_id: postId.value,
      direction,
    });
    if (res.code === 1000) {
      fetchPost();
    } else {
      console.warn(res.msg || 'Vote failed');
    }
  } catch (err: any) {
    if (err.response?.status === 401) {
      router.push('/login');
    } else {
      console.warn('Vote failed');
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
      fetchRemarks();
    } else {
      console.warn(res.msg || 'Failed to post comment');
    }
  } catch (err: any) {
    if (err.response?.status === 401) {
      router.push('/login');
    } else {
      console.warn('Failed to post comment');
    }
  } finally {
    submittingRemark.value = false;
  }
};

onMounted(() => {
  fetchPost();
  fetchRemarks();
  fetchBookmarkStatus();
});
</script>