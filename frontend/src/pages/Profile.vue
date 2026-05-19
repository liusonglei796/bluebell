<template>
  <div class="max-w-7xl mx-auto px-4 pt-24 pb-12">
    <div v-if="loading" class="flex justify-center items-center h-64">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-black"></div>
    </div>
    
    <div v-else-if="error" class="glass rounded-3xl p-8 text-center text-red-500">
      {{ error }}
    </div>

    <div v-else class="grid grid-cols-1 lg:grid-cols-3 gap-8">
      <!-- Left Column: User Profile -->
      <div class="lg:col-span-1">
        <div class="glass rounded-3xl p-8 sticky top-24">
          <div class="flex flex-col items-center text-center">
            <div class="w-32 h-32 bg-black rounded-3xl flex items-center justify-center mb-6 shadow-xl">
              <UserIcon :size="64" class="text-white" />
            </div>
            <h1 class="text-3xl font-black tracking-tighter mb-1">u/{{ profile?.username }}</h1>
            <p class="text-gray-500 font-medium mb-6">User ID: {{ profile?.user_id }}</p>
            
            <div class="grid grid-cols-2 gap-4 w-full mb-8">
              <div class="bg-black/5 p-4 rounded-2xl text-center">
                <div class="text-xl font-black">{{ profile?.reputation || 0 }}</div>
                <div class="text-xs text-gray-500 font-bold uppercase tracking-widest">Reputation</div>
              </div>
              <div class="bg-black/5 p-4 rounded-2xl text-center">
                <div class="text-xl font-black">{{ profile?.post_count || 0 }}</div>
                <div class="text-xs text-gray-500 font-bold uppercase tracking-widest">Posts</div>
              </div>
              <div class="bg-black/5 p-4 rounded-2xl text-center">
                <div class="text-xl font-black">{{ profile?.follower_count || 0 }}</div>
                <div class="text-xs text-gray-500 font-bold uppercase tracking-widest">Followers</div>
              </div>
              <div class="bg-black/5 p-4 rounded-2xl text-center">
                <div class="text-xl font-black">{{ profile?.following_count || 0 }}</div>
                <div class="text-xs text-gray-500 font-bold uppercase tracking-widest">Following</div>
              </div>
            </div>

            <template v-if="!isSelf">
              <button 
                @click="toggleFollow"
                class="w-full py-4 rounded-2xl font-black text-lg transition-all active:scale-95 shadow-lg"
                :class="profile?.is_followed ? 'bg-white border-2 border-black text-black' : 'bg-black text-white hover:bg-gray-800'"
              >
                {{ profile?.is_followed ? 'UNFOLLOW' : 'FOLLOW' }}
              </button>
            </template>
          </div>
        </div>
      </div>

      <!-- Right Column: Activities -->
      <div class="lg:col-span-2">
        <h2 class="text-2xl font-black tracking-tighter mb-6 px-2 italic uppercase">Activity Feed</h2>
        
        <div v-if="activities.length === 0" class="glass rounded-3xl p-12 text-center text-gray-500 font-bold italic">
          No recent activity found.
        </div>
        
        <div v-else class="space-y-4">
          <div v-for="activity in activities" :key="activity.id" class="glass rounded-2xl p-6 hover:-translate-y-1 transition-all duration-300">
            <div class="flex items-start gap-4">
              <div class="p-3 bg-black/5 rounded-xl">
                <component :is="getActivityIcon(activity.type)" :size="20" class="text-black" />
              </div>
              <div class="flex-grow">
                <div class="flex justify-between items-start mb-1">
                  <p class="font-bold text-black">{{ getActivityText(activity) }}</p>
                  <span class="text-xs text-gray-400 font-bold">{{ formatDate(activity.created_at) }}</span>
                </div>
                <p v-if="activity.content" class="text-gray-500 text-sm italic line-clamp-2 mt-2 bg-black/[0.02] p-3 rounded-lg border-l-4 border-black/10">
                  "{{ activity.content }}"
                </p>
                <div v-if="activity.target_id && activity.type !== 'follow'" class="mt-4">
                  <router-link 
                    :to="activity.type === 'post' ? `/post/${activity.target_id}` : '#'"
                    class="text-xs font-black uppercase tracking-widest hover:underline inline-flex items-center gap-1"
                  >
                    View Details <ChevronRightIcon :size="12" />
                  </router-link>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue';
import { useRoute } from 'vue-router';
import { useAuthStore } from '../store/auth';
import request from '../api/request';
import { 
  UserIcon, 
  MessageSquareIcon, 
  ArrowBigUpIcon, 
  UserPlusIcon, 
  FileTextIcon,
  ChevronRightIcon
} from 'lucide-vue-next';

const route = useRoute();
const authStore = useAuthStore();

const profile = ref<any>(null);
const activities = ref<any[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);

const userId = computed(() => route.params.id as string);
const isSelf = computed(() => authStore.user?.user_id === userId.value);

const fetchData = async () => {
  loading.value = true;
  error.value = null;
  try {
    const [profileRes, activitiesRes]: any = await Promise.all([
      request.get(`/user/${userId.value}`),
      request.get(`/user/${userId.value}/activities`)
    ]);

    if (profileRes.code === 1000) {
      profile.value = profileRes.data;
    } else {
      error.value = profileRes.msg || 'Failed to load profile';
    }

    if (activitiesRes.code === 1000) {
      activities.value = activitiesRes.data;
    }
  } catch (err: any) {
    console.error(err);
    error.value = 'Failed to fetch user data. Please try again later.';
  } finally {
    loading.value = false;
  }
};

const toggleFollow = async () => {
  if (!authStore.token) {
    alert('Please login to follow users');
    return;
  }
  
  try {
    const action = profile.value.is_followed ? 'unfollow' : 'follow';
    const res: any = await request.post(`/user/${action}`, {
      user_id: userId.value
    });
    
    if (res.code === 1000) {
      profile.value.is_followed = !profile.value.is_followed;
      profile.value.follower_count += profile.value.is_followed ? 1 : -1;
    }
  } catch (err) {
    alert('Failed to update follow status');
  }
};

const getActivityIcon = (type: string) => {
  switch (type) {
    case 'post': return FileTextIcon;
    case 'comment': return MessageSquareIcon;
    case 'vote': return ArrowBigUpIcon;
    case 'follow': return UserPlusIcon;
    default: return UserIcon;
  }
};

const getActivityText = (activity: any) => {
  switch (activity.type) {
    case 'post': return `Created a new post: ${activity.title}`;
    case 'comment': return `Commented on a post`;
    case 'vote': return `Upvoted a post`;
    case 'follow': return `Followed u/${activity.target_name}`;
    default: return 'Performed an action';
  }
};

const formatDate = (dateStr: string) => {
  const date = new Date(dateStr);
  return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
};

onMounted(fetchData);
watch(() => route.params.id, fetchData);
</script>
