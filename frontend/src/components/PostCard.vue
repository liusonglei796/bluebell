<template>
  <div class="bg-white p-4 border border-gray-200 rounded-lg shadow-sm hover:border-gray-300 transition-colors cursor-pointer" @click="goToPost">
    <div class="flex">
      <div class="flex flex-col items-center mr-4" @click.stop>
        <button @click="vote(1)" class="text-gray-400 hover:text-indigo-600 focus:outline-none">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7"></path></svg>
        </button>
        <span class="font-bold text-gray-700 my-1">{{ post.score || 0 }}</span>
        <button @click="vote(-1)" class="text-gray-400 hover:text-red-600 focus:outline-none">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
        </button>
      </div>
      <div class="flex-grow">
        <div class="text-xs text-gray-500 mb-1">
          Posted by <span class="font-medium">{{ post.author_name }}</span> in 
          <router-link :to="`/community/${post.community_id}`" class="font-medium hover:underline" @click.stop>
            {{ post.community?.name || `Community ${post.community_id}` }}
          </router-link>
        </div>
        <h3 class="text-lg font-bold text-gray-900 mb-2">{{ post.title }}</h3>
        <p class="text-sm text-gray-700 line-clamp-3">{{ post.content }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import request from '../api/request';

const props = defineProps<{
  post: any;
}>();

const router = useRouter();

const goToPost = () => {
  router.push(`/post/${props.post.id}`);
};

const vote = async (direction: number) => {
  try {
    const res: any = await request.post('/vote', {
      post_id: props.post.id,
      direction,
    });
    if (res.code === 1000) {
      alert('Vote successful!');
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
</script>