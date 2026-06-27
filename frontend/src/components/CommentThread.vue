<template>
  <div class="space-y-3">
    
    <div class="glass rounded-2xl p-4 bg-white/40 border-l-4 border-black/20">
      <div class="flex justify-between items-start mb-2">
        <div>
          <span class="font-bold text-black">{{ remark.author_name || `User ${remark.author_id}` }}</span>
          <span class="text-xs text-gray-400 ml-2">{{ formatTime(remark.create_time) }}</span>
        </div>
        <button 
          v-if="canReply"
          @click="() => toggleReplyForm()"
          class="text-xs text-gray-500 hover:text-black transition-colors font-bold"
        >
          {{ showReplyForm ? 'Cancel' : 'Reply' }}
        </button>
      </div>
      
      <p class="text-gray-800 whitespace-pre-wrap text-sm leading-relaxed">{{ remark.content }}</p>

      
      <form v-if="showReplyForm" @submit.prevent="submitReply" class="mt-3 pt-3 border-t border-black/10">
        <textarea 
          v-model="replyContent" 
          rows="2" 
          class="w-full p-2 bg-white/50 border border-black/10 rounded-lg focus:outline-none focus:ring-2 focus:ring-black/10 text-sm" 
          placeholder="Write a reply..."
        ></textarea>
        <div class="mt-2 flex justify-end gap-2">
          <button type="button" @click="() => toggleReplyForm()" class="text-xs px-3 py-1 text-gray-600 hover:text-black">
            Cancel
          </button>
          <button type="submit" :disabled="submittingReply || !replyContent.trim()" class="text-xs px-3 py-1 bg-black text-white rounded-lg font-bold hover:bg-gray-800 disabled:opacity-50">
            {{ submittingReply ? 'Posting...' : 'Reply' }}
          </button>
        </div>
      </form>

      
      <div v-if="replies.length > 0" class="mt-4 ml-4 space-y-3 border-l-2 border-black/10 pl-4">
        <div v-for="reply in replies" :key="reply.id" class="bg-white/30 rounded-lg p-3">
          <div class="flex justify-between items-start mb-1">
            <div>
              <span class="font-bold text-black text-sm">{{ reply.author_name || `User ${reply.author_id}` }}</span>
              <span class="text-xs text-gray-400 ml-2">{{ formatTime(reply.create_time) }}</span>
            </div>
            <button 
              v-if="canReply"
              @click="toggleReplyForm(reply.id)"
              class="text-xs text-gray-500 hover:text-black transition-colors font-bold"
            >
              {{ activeReplyTo === reply.id ? 'Cancel' : 'Reply' }}
            </button>
          </div>
          <p class="text-gray-800 whitespace-pre-wrap text-xs leading-relaxed">{{ reply.content }}</p>

          
          <form v-if="activeReplyTo === reply.id" @submit.prevent="submitReply" class="mt-2 pt-2 border-t border-black/10">
            <textarea 
              v-model="replyContent" 
              rows="2" 
              class="w-full p-2 bg-white/50 border border-black/10 rounded-lg focus:outline-none focus:ring-2 focus:ring-black/10 text-sm" 
              placeholder="Write a reply..."
            ></textarea>
            <div class="mt-2 flex justify-end gap-2">
              <button type="button" @click="activeReplyTo = null" class="text-xs px-3 py-1 text-gray-600 hover:text-black">
                Cancel
              </button>
              <button type="submit" :disabled="submittingReply || !replyContent.trim()" class="text-xs px-3 py-1 bg-black text-white rounded-lg font-bold hover:bg-gray-800 disabled:opacity-50">
                {{ submittingReply ? 'Posting...' : 'Reply' }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { useAuthStore } from '../store/auth';
import request from '../api/request';

const props = defineProps<{
  remark: any;
  postId: string;
}>();

const emit = defineEmits<{
  (e: 'update'): void;
}>();

const authStore = useAuthStore();
const showReplyForm = ref(false);
const activeReplyTo = ref<string | null>(null);
const replyContent = ref('');
const submittingReply = ref(false);
const replies = ref<any[]>([]);

const canReply = computed(() => !!authStore.token);

const formatTime = (timestamp: string) => {
  try {
    return new Date(timestamp).toLocaleDateString('en-US', { 
      month: 'short', 
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  } catch {
    return 'Unknown';
  }
};

const toggleReplyForm = (replyId?: string) => {
  if (replyId) {
    activeReplyTo.value = activeReplyTo.value === replyId ? null : replyId;
    showReplyForm.value = false;
  } else {
    showReplyForm.value = !showReplyForm.value;
    activeReplyTo.value = null;
  }
  if (!showReplyForm.value && activeReplyTo.value === null) {
    replyContent.value = '';
  }
};

const submitReply = async () => {
  if (!replyContent.value.trim()) return;
  submittingReply.value = true;
  
  try {
    const res: any = await request.post('/remark', {
      post_id: props.postId,
      content: replyContent.value,
      reply_to: activeReplyTo.value || props.remark.id,
    });
    
    if (res.code === 200) {
      replyContent.value = '';
      showReplyForm.value = false;
      activeReplyTo.value = null;
      emit('update');
    }
  } catch (err) {
    console.error('Failed to submit reply', err);
  } finally {
    submittingReply.value = false;
  }
};

const loadReplies = async () => {
  try {
    const res: any = await request.get(`/post/${props.postId}/remarks?reply_to=${props.remark.id}`);
    if (res.code === 200) {
      replies.value = res.data || [];
    }
  } catch (err) {
    console.error('Failed to load replies', err);
  }
};

loadReplies();
</script>
