<template>
  <div class="glass rounded-[24px] p-6 hover:-translate-y-1 hover:shadow-md transition-all duration-300 cursor-pointer" @click="goToPost">
    <div class="flex gap-6">
      <!-- Voting Sidebar -->
      <div class="flex flex-col items-center bg-black/5 rounded-2xl p-2 h-fit" @click.stop>
        <button @click="vote(1)" class="text-gray-400 hover:text-black transition-colors focus:outline-none p-1 cursor-pointer">
          <ArrowBigUpIcon :stroke-width="2.5" class="w-6 h-6" />
        </button>
        <span class="font-bold text-black my-1 text-base font-heading">{{ post.vote_num || 0 }}</span>
        <button @click="vote(-1)" class="text-gray-400 hover:text-black transition-colors focus:outline-none p-1 cursor-pointer">
          <ArrowBigDownIcon :stroke-width="2.5" class="w-6 h-6" />
        </button>
      </div>

      <!-- Content -->
      <div class="flex-grow">
        <div class="flex items-center gap-2 text-xs text-gray-500 mb-2">
          <span class="bg-black/5 px-2 py-0.5 rounded-full text-black font-bold border border-black/5">
            <router-link :to="`/community/${post.community_id}`" class="hover:underline" @click.stop>
              c/{{ post.community?.name || `Community ${post.community_id}` }}
            </router-link>
          </span>
          <span>•</span>
          <span>Posted by 
            <router-link :to="`/user/${post.author_id}`" class="font-medium text-black hover:underline" @click.stop>
              u/{{ post.author_name }}
            </router-link>
          </span>
        </div>
        <h3 class="text-xl font-black text-black mb-2 leading-tight tracking-tight font-heading">{{ post.title }}</h3>
        <p class="text-gray-600 line-clamp-3 leading-relaxed text-sm">{{ post.content }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import { ArrowBigUpIcon, ArrowBigDownIcon } from 'lucide-vue-next';
import request from '../api/request';

export interface Post {
  id: string;
  title: string;
  content: string;
  vote_num: number;
  community_id: number;
  author_id: string;
  author_name: string;
  community?: {
    id: number;
    name: string;
  };
}

const props = defineProps<{
  post: Post;
}>();

const emit = defineEmits<{
  (e: 'voted'): void;
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
      emit('voted');
    } else {
      console.warn(res.msg || 'Vote failed');
    }
  } catch (err: any) {
    if (err.response?.status === 401) {
      router.push('/login');
    } else {
      console.error('Vote failed', err);
    }
  }
};
</script>