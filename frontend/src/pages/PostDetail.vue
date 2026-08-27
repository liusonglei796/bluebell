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

    <!-- Comments Section -->
    <div v-if="post" class="mt-8">
      <h2 class="text-xl font-bold text-gray-900 mb-4">Comments ({{ totalComments }})</h2>
      
      <!-- Add Comment -->
      <form @submit.prevent="submitComment" class="mb-8">
        <textarea v-model="newComment" rows="3" class="w-full p-3 border border-gray-300 rounded-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500" placeholder="What are your thoughts?"></textarea>
        <div class="mt-2 flex justify-end">
          <button type="submit" :disabled="submittingComment || !newComment.trim()" class="px-4 py-2 bg-indigo-600 text-white rounded-md text-sm font-medium hover:bg-indigo-700 disabled:opacity-50">
            {{ submittingComment ? 'Posting...' : 'Comment' }}
          </button>
        </div>
      </form>

      <!-- Comment List -->
      <div class="space-y-4">
        <div v-for="comment in comments" :key="comment.id" class="bg-white p-4 border border-gray-200 rounded-lg shadow-sm">
          <div class="text-sm text-gray-500 mb-2">
            <span class="font-medium text-gray-900">{{ comment.author_name || 'User ' + comment.author_id }}</span>
            &bull; {{ new Date(comment.created_at).toLocaleString() }}
          </div>
          <p class="text-gray-800 whitespace-pre-wrap">{{ comment.content }}</p>

          <!-- Sub Replies Preview -->
          <div v-if="comment.sub_replies && comment.sub_replies.length > 0" class="mt-3 pl-4 border-l-2 border-gray-100 space-y-2 bg-gray-50 p-3 rounded">
            <div v-for="reply in comment.sub_replies" :key="reply.id" class="text-sm">
              <span class="font-medium text-gray-900">{{ reply.author_name }}: </span>
              <span class="text-gray-700">{{ reply.content }}</span>
            </div>
          </div>
        </div>
        <div v-if="comments.length === 0" class="text-gray-500 text-sm">
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
const comments = ref<any[]>([]);
const totalComments = ref(0);
const loading = ref(true);
const newComment = ref('');
const submittingComment = ref(false);

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

const fetchComments = async () => {
  try {
    const res: any = await request.get('/comments', {
      params: {
        post_id: postId.value,
        page: 1,
        size: 50,
        order: 'hot',
      },
    });
    if (res.code === 1000 && res.data) {
      comments.value = res.data.comments || [];
      totalComments.value = res.data.total || 0;
    }
  } catch (err) {
    console.error('Failed to fetch comments', err);
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

const submitComment = async () => {
  if (!newComment.value.trim()) return;
  submittingComment.value = true;
  try {
    const res: any = await request.post('/comment', {
      post_id: postId.value,
      content: newComment.value,
      root_id: 0,
      parent_id: 0,
      reply_to_uid: 0,
    });
    if (res.code === 1000) {
      newComment.value = '';
      fetchComments(); // Refresh comments
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
    submittingComment.value = false;
  }
};

onMounted(() => {
  fetchPost();
  fetchComments();
});
</script>