# Frontend UI Refactoring to Sleek Glassmorphism Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor the frontend UI elements of Bluebell to a premium light-themed monochrome glassmorphism aesthetic with Outfit and Inter font pairings.

**Architecture:** Leverage global CSS variables and class overrides inside `style.css` to build high-fidelity frosted glass effects, utilizing CSS hardware acceleration (`will-change`, `translateZ(0)`) to eliminate hover rendering flickers. Then component-by-component, update templates and styles to fit this spec.

**Tech Stack:** Vue 3, Vite, TypeScript, Tailwind CSS, Lucide icons.

---

### Task 1: Global CSS & Typography Setup

**Files:**
- Modify: `frontend/src/style.css`

- [ ] **Step 1: Write the updated style.css**
  Replace the entire content of [style.css](file:///D:/download/project/bluebell/frontend/src/style.css) with:
  ```css
  @import "tailwindcss";

  @import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Outfit:wght@400;500;700;900&display=swap');

  :root {
    --glass-bg: rgba(255, 255, 255, 0.45);
    --glass-border: rgba(255, 255, 255, 0.25);
    --glass-shadow: 0 8px 32px 0 rgba(31, 38, 135, 0.05);
    --glass-shadow-hover: 0 12px 40px 0 rgba(31, 38, 135, 0.1);
    
    --accent: #000000;
    
    --transition-speed: 0.3s;
    --transition-bounce: cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  .glass {
    background: var(--glass-bg);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    border: 1px solid var(--glass-border);
    box-shadow: var(--glass-shadow);
    
    /* Force GPU rendering from the start to prevent white flashes */
    will-change: transform, backdrop-filter;
    transform: translateZ(0);
    backface-visibility: hidden;
  }

  body {
    margin: 0;
    font-family: 'Inter', -apple-system, sans-serif;
    background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
    background-attachment: fixed;
    min-height: 100vh;
    color: #1d1d1f;
  }

  h1, h2, h3, h4, .font-heading {
    font-family: 'Outfit', sans-serif;
    font-weight: 700;
  }
  ```

- [ ] **Step 2: Run typecheck to verify styles compilation**
  Run: `npm --prefix frontend run typecheck`
  Expected: Command completes successfully.

- [ ] **Step 3: Commit**
  ```bash
  git add frontend/src/style.css
  git commit -m "style: set up global glassmorphic design tokens and fonts"
  ```

---

### Task 2: Navbar Component Refactoring

**Files:**
- Modify: `frontend/src/components/Navbar.vue`

- [ ] **Step 1: Replace Navbar.vue with sleek glassmorphism**
  Replace [Navbar.vue](file:///D:/download/project/bluebell/frontend/src/components/Navbar.vue) with:
  ```vue
  <template>
    <nav class="fixed top-4 left-0 right-0 z-50 px-4">
      <div class="max-w-7xl mx-auto glass rounded-full shadow-lg" style="background: rgba(255, 255, 255, 0.6);">
        <div class="px-6">
          <div class="flex justify-between h-16">
            <div class="flex">
              <router-link to="/" class="flex-shrink-0 flex items-center">
                <span class="text-xl font-black tracking-tighter text-black font-heading">BLUEBELL</span>
              </router-link>
              
              <!-- Search Bar -->
              <div class="hidden sm:ml-8 sm:flex sm:items-center">
                <form @submit.prevent="handleSearch" class="relative">
                  <input 
                    v-model="searchKeyword"
                    type="text" 
                    class="block w-64 pl-4 pr-10 py-2 border border-black/10 rounded-full leading-5 bg-white/50 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-black/20 sm:text-sm backdrop-blur-sm transition-all focus:w-80" 
                    placeholder="Search Bluebell"
                  >
                  <button type="submit" class="absolute inset-y-0 right-0 pr-3 flex items-center">
                    <svg class="h-4 w-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path></svg>
                  </button>
                </form>
              </div>
            </div>
            <div class="flex items-center space-x-6">
              <template v-if="authStore.token">
                <router-link :to="`/user/${authStore.user?.user_id}`" class="text-sm font-semibold text-black hover:underline">
                  Welcome, {{ authStore.user?.username }}
                </router-link>
                <router-link
                  v-if="authStore.isAdmin()"
                  to="/create-community"
                  class="text-sm font-bold text-gray-600 hover:text-black transition-colors"
                >
                  New Community
                </router-link>
                <button @click="logout" class="text-sm font-bold text-gray-500 hover:text-black transition-colors cursor-pointer">
                  Logout
                </button>
                <router-link to="/create-post" class="bg-black text-white px-6 py-2 rounded-full text-sm font-bold shadow-md hover:bg-gray-800 transition-all active:scale-95">
                  New Post
                </router-link>
              </template>
              <template v-else>
                <router-link to="/login" class="text-sm font-bold text-gray-500 hover:text-black transition-colors">
                  Login
                </router-link>
                <router-link to="/signup" class="bg-black text-white px-6 py-2 rounded-full text-sm font-bold shadow-md hover:bg-gray-800 transition-all active:scale-95">
                  Sign up
                </router-link>
              </template>
            </div>
          </div>
        </div>
      </div>
    </nav>
  </template>

  <script setup lang="ts">
  import { ref } from 'vue';
  import { useAuthStore } from '../store/auth';
  import { useRouter } from 'vue-router';

  const authStore = useAuthStore();
  const router = useRouter();
  const searchKeyword = ref('');

  const handleSearch = () => {
    if (searchKeyword.value.trim()) {
      router.push({
        path: '/search',
        query: { q: searchKeyword.value.trim() }
      });
      searchKeyword.value = '';
    }
  };

  const logout = () => {
    authStore.clearAuth();
    router.push('/login');
  };
  </script>
  ```

- [ ] **Step 2: Run typecheck to verify Navbar component**
  Run: `npm --prefix frontend run typecheck`
  Expected: Pass.

- [ ] **Step 3: Commit**
  ```bash
  git add frontend/src/components/Navbar.vue
  git commit -m "feat: refactor Navbar with sleek glassmorphism design and search sizing transition"
  ```

---

### Task 3: PostCard Component Refactoring

**Files:**
- Modify: `frontend/src/components/PostCard.vue`

- [ ] **Step 1: Replace PostCard.vue content**
  Update [PostCard.vue](file:///D:/download/project/bluebell/frontend/src/components/PostCard.vue) to incorporate Outfit fonts, soft transitions, and clean voting block:
  ```vue
  <template>
    <div class="glass rounded-[24px] p-6 hover:-translate-y-1 hover:shadow-md transition-all duration-300 cursor-pointer" @click="goToPost">
      <div class="flex gap-6">
        <!-- Voting Sidebar -->
        <div class="flex flex-col items-center bg-black/5 rounded-2xl p-2 h-fit" @click.stop>
          <button @click="vote(1)" class="text-gray-400 hover:text-black transition-colors focus:outline-none p-1 cursor-pointer">
            <ArrowBigUpIcon :stroke-width="2.5" class="w-6 h-6" />
          </button>
          <span class="font-bold text-black my-1 text-base font-heading">{{ post.score || 0 }}</span>
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
  ```

- [ ] **Step 2: Run typecheck to verify PostCard**
  Run: `npm --prefix frontend run typecheck`
  Expected: Pass.

- [ ] **Step 3: Commit**
  ```bash
  git add frontend/src/components/PostCard.vue
  git commit -m "feat: refactor PostCard to support sleek glass layout and Outfit font"
  ```

---

### Task 4: Home Page Refactoring

**Files:**
- Modify: `frontend/src/pages/Home.vue`

- [ ] **Step 1: Replace Home.vue**
  Replace [Home.vue](file:///D:/download/project/bluebell/frontend/src/pages/Home.vue) to styled glass list layout, hot/new toggle styling, sidebar:
  ```vue
  <template>
    <div class="flex flex-col md:flex-row gap-8">
      <div class="flex-grow space-y-6 md:w-2/3">
        <div class="flex justify-between items-center mb-4">
          <h1 class="text-3xl font-black text-black tracking-tighter font-heading">POPULAR POSTS</h1>
          <div class="flex p-1 bg-white/50 backdrop-blur-md rounded-xl border border-black/5">
            <button @click="changeOrder('score')" 
                    :class="['px-4 py-1.5 text-sm font-bold rounded-lg transition-all duration-200 cursor-pointer', 
                             order === 'score' ? 'bg-black text-white shadow-sm font-heading' : 'text-gray-500 hover:text-black font-heading']">
              HOT
            </button>
            <button @click="changeOrder('time')" 
                    :class="['px-4 py-1.5 text-sm font-bold rounded-lg transition-all duration-200 cursor-pointer', 
                             order === 'time' ? 'bg-black text-white shadow-sm font-heading' : 'text-gray-500 hover:text-black font-heading']">
              NEW
            </button>
          </div>
        </div>
        
        <div v-if="loading" class="flex flex-col items-center justify-center py-20 space-y-4">
          <div class="w-12 h-12 border-4 border-gray-200 border-t-black rounded-full animate-spin"></div>
          <span class="text-black/60 font-bold uppercase tracking-widest text-xs font-heading">Loading posts...</span>
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
          <h2 class="text-xl font-black text-black mb-6 flex items-center gap-2 tracking-tight font-heading">
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
          
          <button class="w-full mt-6 py-4 px-4 rounded-2xl bg-black text-white font-black hover:bg-gray-800 transition-all duration-300 shadow-lg active:scale-95 uppercase tracking-wider text-sm font-heading cursor-pointer">
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
  ```

- [ ] **Step 2: Run typecheck to verify Home.vue**
  Run: `npm --prefix frontend run typecheck`
  Expected: Pass.

- [ ] **Step 3: Commit**
  ```bash
  git add frontend/src/pages/Home.vue
  git commit -m "feat: update Home feed layouts and communities sidebar styles"
  ```

---

### Task 5: User Profile Page Refactoring

**Files:**
- Modify: `frontend/src/pages/Profile.vue`

- [ ] **Step 1: Replace Profile.vue content**
  Update [Profile.vue](file:///D:/download/project/bluebell/frontend/src/pages/Profile.vue) to incorporate glassmorphism, Outfit statistic cards, and custom Activity list logs:
  ```vue
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
              <h1 class="text-3xl font-black tracking-tighter mb-1 font-heading">u/{{ profile?.username }}</h1>
              <p class="text-gray-500 font-medium mb-6">User ID: {{ profile?.user_id }}</p>
              
              <div class="grid grid-cols-2 gap-4 w-full mb-8">
                <div class="bg-black/5 p-4 rounded-2xl text-center">
                  <div class="text-xl font-black font-heading">{{ profile?.reputation || 0 }}</div>
                  <div class="text-xs text-gray-500 font-bold uppercase tracking-widest font-heading">Reputation</div>
                </div>
                <div class="bg-black/5 p-4 rounded-2xl text-center">
                  <div class="text-xl font-black font-heading">{{ profile?.post_count || 0 }}</div>
                  <div class="text-xs text-gray-500 font-bold uppercase tracking-widest font-heading">Posts</div>
                </div>
                <div class="bg-black/5 p-4 rounded-2xl text-center">
                  <div class="text-xl font-black font-heading">{{ profile?.follower_count || 0 }}</div>
                  <div class="text-xs text-gray-500 font-bold uppercase tracking-widest font-heading">Followers</div>
                </div>
                <div class="bg-black/5 p-4 rounded-2xl text-center">
                  <div class="text-xl font-black font-heading">{{ profile?.following_count || 0 }}</div>
                  <div class="text-xs text-gray-500 font-bold uppercase tracking-widest font-heading">Following</div>
                </div>
              </div>

              <template v-if="!isSelf">
                <button 
                  @click="toggleFollow"
                  class="w-full py-4 rounded-2xl font-black text-lg transition-all active:scale-95 shadow-lg font-heading cursor-pointer"
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
          <h2 class="text-2xl font-black tracking-tighter mb-6 px-2 italic uppercase font-heading">Activity Feed</h2>
          
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
                    <p class="font-bold text-black font-heading">{{ getActivityText(activity) }}</p>
                    <span class="text-xs text-gray-400 font-bold font-heading">{{ formatDate(activity.created_at) }}</span>
                  </div>
                  <p v-if="activity.content" class="text-gray-500 text-sm italic line-clamp-2 mt-2 bg-black/[0.02] p-3 rounded-lg border-l-4 border-black/10">
                    "{{ activity.content }}"
                  </p>
                  <div v-if="activity.target_id && activity.type !== 'follow'" class="mt-4">
                    <router-link 
                      :to="activity.type === 'post' ? `/post/${activity.target_id}` : '#'"
                      class="text-xs font-black uppercase tracking-widest hover:underline inline-flex items-center gap-1 font-heading"
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
  ```

- [ ] **Step 2: Run typecheck to verify Profile.vue**
  Run: `npm --prefix frontend run typecheck`
  Expected: Pass.

- [ ] **Step 3: Commit**
  ```bash
  git add frontend/src/pages/Profile.vue
  git commit -m "feat: refactor Profile stats cards and activity lists to glassmorphic layout"
  ```

---

### Task 6: Post Details & Remarks Page Refactoring

**Files:**
- Modify: `frontend/src/pages/PostDetail.vue`

- [ ] **Step 1: Replace PostDetail.vue content**
  Update [PostDetail.vue](file:///D:/download/project/bluebell/frontend/src/pages/PostDetail.vue) to incorporate Outfit header text, glass content boxes, clean comment textareas:
  ```vue
  <template>
    <div class="max-w-4xl mx-auto pt-8">
      <div v-if="loading" class="text-center py-10">
        <span class="text-gray-500">Loading post...</span>
      </div>

      <div v-else-if="post" class="glass p-6 rounded-3xl">
        <div class="flex">
          <div class="flex flex-col items-center mr-6 bg-black/5 rounded-2xl p-2 h-fit">
            <button @click="vote(1)" class="text-gray-400 hover:text-black focus:outline-none p-1 cursor-pointer">
              <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7"></path></svg>
            </button>
            <span class="text-xl font-bold text-black my-2 font-heading">{{ post.score || 0 }}</span>
            <button @click="vote(-1)" class="text-gray-400 hover:text-black focus:outline-none p-1 cursor-pointer">
              <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
            </button>
            <div class="mt-4 pt-4 border-t border-black/10 w-full flex justify-center">
              <button @click="toggleBookmark" class="text-gray-400 hover:text-yellow-500 focus:outline-none cursor-pointer" :title="isBookmarked ? 'Remove bookmark' : 'Add bookmark'">
                <svg class="w-6 h-6" :fill="isBookmarked ? 'currentColor' : 'none'" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z"></path></svg>
              </button>
            </div>
            <span v-if="bookmarkCount > 0" class="text-xs text-gray-500 mt-1 block font-heading">{{ bookmarkCount }}</span>
          </div>
          <div class="flex-grow">
            <div class="text-sm text-gray-500 mb-2">
              Posted by <span class="font-medium text-gray-900 font-heading">u/{{ post.author_name }}</span>
              in <router-link :to="`/community/${post.community_id}`" class="font-medium text-black hover:underline font-heading">c/{{ post.community?.name || `Community ${post.community_id}` }}</router-link>
            </div>
            <h1 class="text-2xl font-bold text-gray-900 mb-4 font-heading">{{ post.title }}</h1>
            <div class="prose max-w-none text-gray-800 whitespace-pre-wrap leading-relaxed">
              {{ post.content }}
            </div>
          </div>
        </div>
      </div>

      <div v-else class="text-center py-10">
        <span class="text-gray-500">Post not found.</span>
      </div>

      <!-- Remarks Section -->
      <div v-if="post" class="mt-8">
        <h2 class="text-xl font-bold text-gray-900 mb-4 font-heading">Comments</h2>

        <!-- Add Comment -->
        <form @submit.prevent="submitRemark" class="mb-8">
          <textarea v-model="newRemark" rows="3" class="w-full p-3 border border-black/10 rounded-2xl focus:outline-none focus:ring-2 focus:ring-black/20 bg-white/50 backdrop-blur-sm" placeholder="What are your thoughts?"></textarea>
          <div class="mt-2 flex justify-end">
            <button type="submit" :disabled="submittingRemark || !newRemark.trim()" class="px-6 py-2 bg-black text-white rounded-full text-sm font-bold hover:bg-gray-800 disabled:opacity-50 font-heading cursor-pointer">
              {{ submittingRemark ? 'Posting...' : 'Comment' }}
            </button>
          </div>
        </form>

        <!-- Comment List -->
        <div class="space-y-4">
          <div v-for="remark in remarks" :key="remark.id" class="glass p-4 rounded-2xl">
            <div class="text-sm text-gray-500 mb-2">
              <span class="font-medium text-gray-900 font-heading">u/{{ remark.author_name || 'User ' + remark.author_id }}</span>
              &bull; <span class="font-medium text-gray-400">{{ new Date(remark.create_time).toLocaleString() }}</span>
            </div>
            <p class="text-gray-800 whitespace-pre-wrap text-sm leading-relaxed">{{ remark.content }}</p>
          </div>
          <div v-if="remarks.length === 0" class="text-gray-500 text-sm">
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
  import { bookmarkAPI } from '../api/bookmark';
  import { useAuthStore } from '../store/auth';

  const route = useRoute();
  const router = useRouter();
  const authStore = useAuthStore();
  const postId = ref(Number(route.params.id));
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
        alert('Vote successful!');
        fetchPost();
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
        alert(res.msg || 'Failed to post comment');
      }
    } catch (err: any) {
      if (err.response?.status === 401) {
        router.push('/login');
      } else {
        alert('Failed to post comment');
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
  ```

- [ ] **Step 2: Run typecheck to verify PostDetail.vue**
  Run: `npm --prefix frontend run typecheck`
  Expected: Pass.

- [ ] **Step 3: Commit**
  ```bash
  git add frontend/src/pages/PostDetail.vue
  git commit -m "feat: refactor PostDetail page to support glass cards and layout structures"
  ```

---

### Task 7: Forms & Search Pages Refactoring

**Files:**
- Modify: `frontend/src/pages/CreatePost.vue`
- Modify: `frontend/src/pages/CreateCommunity.vue`
- Modify: `frontend/src/pages/Search.vue`

- [ ] **Step 1: Refactor CreatePost.vue layout**
  Modify [CreatePost.vue](file:///D:/download/project/bluebell/frontend/src/pages/CreatePost.vue) inputs and containers:
  ```vue
  <template>
    <div class="max-w-2xl mx-auto glass p-8 rounded-3xl mt-8">
      <h2 class="text-2xl font-bold mb-6 text-gray-900 font-heading">Create a New Post</h2>
      <form @submit.prevent="submitPost" class="space-y-6">
        <div>
          <label for="community" class="block text-sm font-medium text-gray-700 font-heading">Community</label>
          <select v-model="form.community_id" id="community" required class="mt-1 block w-full pl-3 pr-10 py-2.5 text-base border border-black/10 rounded-2xl bg-white/50 focus:outline-none focus:ring-2 focus:ring-black/20 sm:text-sm shadow-sm">
            <option disabled value="">Select a community</option>
            <option v-for="community in communities" :key="community.id" :value="community.id">
              {{ community.name }}
            </option>
          </select>
        </div>
        <div>
          <label for="title" class="block text-sm font-medium text-gray-700 font-heading">Title</label>
          <input v-model="form.title" type="text" id="title" required class="mt-1 block w-full px-4 py-2.5 border border-black/10 rounded-2xl bg-white/50 focus:outline-none focus:ring-2 focus:ring-black/20 sm:text-sm" />
        </div>
        <div>
          <label for="content" class="block text-sm font-medium text-gray-700 font-heading">Content</label>
          <textarea v-model="form.content" id="content" rows="6" required class="mt-1 block w-full px-4 py-2.5 border border-black/10 rounded-2xl bg-white/50 focus:outline-none focus:ring-2 focus:ring-black/20 sm:text-sm"></textarea>
        </div>
        <div v-if="errorMsg" class="text-red-600 text-sm">
          {{ errorMsg }}
        </div>
        <div class="flex justify-end">
          <button type="submit" :disabled="loading" class="bg-black text-white px-8 py-3 rounded-full text-sm font-bold shadow-md hover:bg-gray-800 transition-all active:scale-95 font-heading cursor-pointer disabled:opacity-50">
            {{ loading ? 'Posting...' : 'Post' }}
          </button>
        </div>
      </form>
    </div>
  </template>

  <script setup lang="ts">
  import { ref, onMounted } from 'vue';
  import { useRouter } from 'vue-router';
  import request from '../api/request';

  const communities = ref<any[]>([]);
  const form = ref({
    community_id: '',
    title: '',
    content: '',
  });
  const loading = ref(false);
  const errorMsg = ref('');
  const router = useRouter();

  const fetchCommunities = async () => {
    try {
      const res: any = await request.get('/community');
      if (res.code === 1000) {
        communities.value = res.data || [];
      }
    } catch (err) {
      console.error('Failed to fetch communities', err);
    }
  };

  const submitPost = async () => {
    loading.value = true;
    errorMsg.value = '';
    try {
      const payload = {
        ...form.value,
        community_id: parseInt(form.value.community_id as string, 10),
      };
      const res: any = await request.post('/post', payload);
      if (res.code === 1000) {
        router.push('/');
      } else {
        errorMsg.value = res.msg || 'Failed to create post';
      }
    } catch (err: any) {
      errorMsg.value = err.response?.data?.msg || 'An error occurred';
    } finally {
      loading.value = false;
    }
  };

  onMounted(() => {
    fetchCommunities();
  });
  </script>
  ```

- [ ] **Step 2: Refactor CreateCommunity.vue layout**
  Modify [CreateCommunity.vue](file:///D:/download/project/bluebell/frontend/src/pages/CreateCommunity.vue) fields and button:
  ```vue
  <template>
    <div class="max-w-lg mx-auto glass p-8 rounded-3xl mt-16">
      <h2 class="text-2xl font-bold mb-6 text-center text-gray-900 font-heading">Create New Community</h2>
      <form @submit.prevent="handleCreate" class="space-y-4">
        <div>
          <label for="name" class="block text-sm font-medium text-gray-700 font-heading">Community Name</label>
          <input
            v-model="form.name"
            type="text"
            id="name"
            required
            placeholder="e.g., Go, Python, React"
            class="mt-1 block w-full px-4 py-2.5 border border-black/10 rounded-2xl bg-white/50 focus:outline-none focus:ring-2 focus:ring-black/20 sm:text-sm"
          />
        </div>
        <div>
          <label for="introduction" class="block text-sm font-medium text-gray-700 font-heading">Introduction</label>
          <textarea
            v-model="form.introduction"
            id="introduction"
            required
            rows="4"
            placeholder="Brief description of this community..."
            class="mt-1 block w-full px-4 py-2.5 border border-black/10 rounded-2xl bg-white/50 focus:outline-none focus:ring-2 focus:ring-black/20 sm:text-sm"
          ></textarea>
        </div>
        <div v-if="errorMsg" class="text-red-600 text-sm">
          {{ errorMsg }}
        </div>
        <div v-if="successMsg" class="text-green-600 text-sm">
          {{ successMsg }}
        </div>
        <button
          type="submit"
          :disabled="loading"
          class="w-full bg-black text-white py-3 rounded-full text-sm font-bold shadow-md hover:bg-gray-800 transition-all active:scale-95 font-heading cursor-pointer disabled:opacity-50"
        >
          {{ loading ? 'Creating...' : 'Create Community' }}
        </button>
      </form>
      <div class="mt-4 text-center text-sm">
        <router-link to="/" class="font-bold text-gray-500 hover:text-black transition-colors font-heading">Back to Home</router-link>
      </div>
    </div>
  </template>

  <script setup lang="ts">
  import { ref } from 'vue';
  import { useRouter } from 'vue-router';
  import { useAuthStore } from '../store/auth';
  import request from '../api/request';

  const form = ref({ name: '', introduction: '' });
  const loading = ref(false);
  const errorMsg = ref('');
  const successMsg = ref('');
  const router = useRouter();
  const authStore = useAuthStore();

  if (!authStore.isAdmin()) {
    errorMsg.value = 'You do not have permission to create a community.';
  }

  const handleCreate = async () => {
    if (!authStore.isAdmin()) {
      errorMsg.value = 'You do not have permission to create a community.';
      return;
    }

    loading.value = true;
    errorMsg.value = '';
    successMsg.value = '';

    try {
      const res: any = await request.post('/community', form.value);
      if (res.code === 1000) {
        successMsg.value = 'Community created successfully!';
        form.value = { name: '', introduction: '' };
        setTimeout(() => {
          router.push('/');
        }, 1500);
      } else {
        errorMsg.value = res.msg || 'Failed to create community';
      }
    } catch (err: any) {
      errorMsg.value = err.response?.data?.msg || 'An error occurred';
    } finally {
      loading.value = false;
    }
  };
  </script>
  ```

- [ ] **Step 3: Refactor Search.vue layout**
  Update [Search.vue](file:///D:/download/project/bluebell/frontend/src/pages/Search.vue) heading tags:
  ```vue
  <template>
    <div class="max-w-4xl mx-auto space-y-6">
      <div class="flex items-center justify-between mb-6">
        <h1 class="text-3xl font-black text-black tracking-tighter font-heading">SEARCH RESULTS</h1>
        <span class="text-sm text-gray-500 font-bold uppercase tracking-widest font-heading">
          Query: "{{ keyword }}"
        </span>
      </div>

      <div v-if="loading" class="flex flex-col items-center justify-center py-20 space-y-4">
        <div class="w-12 h-12 border-4 border-gray-200 border-t-black rounded-full animate-spin"></div>
        <span class="text-black/60 font-bold uppercase tracking-widest text-xs font-heading">Searching posts...</span>
      </div>

      <template v-else>
        <PostCard v-for="post in posts" :key="post.id" :post="post" />
        <div v-if="posts.length === 0" class="glass rounded-[24px] p-12 text-center text-gray-500 font-bold italic">
          No matches found for "{{ keyword }}" in this universe.
        </div>
      </template>
    </div>
  </template>

  <script setup lang="ts">
  import { ref, watch, onMounted } from 'vue';
  import { useRoute } from 'vue-router';
  import request from '../api/request';
  import PostCard from '../components/PostCard.vue';

  const route = useRoute();
  const posts = ref<any[]>([]);
  const loading = ref(false);
  const keyword = ref('');

  const search = async () => {
    const q = route.query.q as string;
    if (!q) return;
    
    keyword.value = q;
    loading.value = true;
    try {
      const res: any = await request.get(`/posts2?search=${encodeURIComponent(q)}&page=1&size=20`);
      if (res.code === 1000) {
        posts.value = res.data || [];
      }
    } catch (err) {
      console.error('Search failed', err);
    } finally {
      loading.value = false;
    }
  };

  onMounted(search);
  watch(() => route.query.q, search);
  </script>
  ```

- [ ] **Step 4: Run typecheck to verify form and search components**
  Run: `npm --prefix frontend run typecheck`
  Expected: Pass.

- [ ] **Step 5: Commit**
  ```bash
  git add frontend/src/pages/CreatePost.vue frontend/src/pages/CreateCommunity.vue frontend/src/pages/Search.vue
  git commit -m "feat: refactor CreatePost, CreateCommunity, and Search pages to glass card designs"
  ```

---

### Task 8: Authentication Pages Refactoring

**Files:**
- Modify: `frontend/src/pages/Login.vue`
- Modify: `frontend/src/pages/Signup.vue`

- [ ] **Step 1: Replace Login.vue content**
  Update [Login.vue](file:///D:/download/project/bluebell/frontend/src/pages/Login.vue) to glass container, clean inputs, and Outfit headers:
  ```vue
  <template>
    <div class="max-w-md mx-auto glass p-8 rounded-3xl mt-16">
      <h2 class="text-3xl font-black mb-6 text-center text-gray-900 font-heading">LOGIN</h2>
      <form @submit.prevent="handleLogin" class="space-y-4">
        <div>
          <label for="username" class="block text-sm font-bold text-gray-700 font-heading">USERNAME</label>
          <input
            v-model="form.username"
            type="text"
            id="username"
            required
            class="mt-1 block w-full px-4 py-2.5 border border-black/10 rounded-2xl bg-white/50 focus:outline-none focus:ring-2 focus:ring-black/20 sm:text-sm"
          />
        </div>
        <div>
          <label for="password" class="block text-sm font-bold text-gray-700 font-heading">PASSWORD</label>
          <input
            v-model="form.password"
            type="password"
            id="password"
            required
            class="mt-1 block w-full px-4 py-2.5 border border-black/10 rounded-2xl bg-white/50 focus:outline-none focus:ring-2 focus:ring-black/20 sm:text-sm"
          />
        </div>
        <div v-if="errorMsg" class="text-red-600 text-sm">
          {{ errorMsg }}
        </div>
        <button
          type="submit"
          :disabled="loading"
          class="w-full bg-black text-white py-3 rounded-full text-sm font-bold shadow-md hover:bg-gray-800 transition-all active:scale-95 font-heading cursor-pointer disabled:opacity-50"
        >
          {{ loading ? 'LOGGING IN...' : 'LOGIN' }}
        </button>
      </form>
      <div class="mt-6 text-center text-sm">
        <span class="text-gray-500">Need an account?</span>
        <router-link to="/signup" class="ml-1 font-bold text-black hover:underline font-heading">SIGN UP</router-link>
      </div>
    </div>
  </template>

  <script setup lang="ts">
  import { ref } from 'vue';
  import { useRouter } from 'vue-router';
  import { useAuthStore } from '../store/auth';
  import request from '../api/request';

  const form = ref({ username: '', password: '' });
  const loading = ref(false);
  const errorMsg = ref('');
  const router = useRouter();
  const authStore = useAuthStore();

  const handleLogin = async () => {
    loading.value = true;
    errorMsg.value = '';
    try {
      const res: any = await request.post('/login', form.value);
      if (res.code === 1000) {
        authStore.setAuth(res.data.token, {
          user_id: res.data.user_id,
          username: res.data.user_name,
        });
        router.push('/');
      } else {
        errorMsg.value = res.msg || 'Login failed';
      }
    } catch (err: any) {
      errorMsg.value = err.response?.data?.msg || 'An error occurred';
    } finally {
      loading.value = false;
    }
  };
  </script>
  ```

- [ ] **Step 2: Replace Signup.vue content**
  Update [Signup.vue](file:///D:/download/project/bluebell/frontend/src/pages/Signup.vue):
  ```vue
  <template>
    <div class="max-w-md mx-auto glass p-8 rounded-3xl mt-16">
      <h2 class="text-3xl font-black mb-6 text-center text-gray-900 font-heading">SIGN UP</h2>
      <form @submit.prevent="handleSignup" class="space-y-4">
        <div>
          <label for="username" class="block text-sm font-bold text-gray-700 font-heading">USERNAME</label>
          <input
            v-model="form.username"
            type="text"
            id="username"
            required
            class="mt-1 block w-full px-4 py-2.5 border border-black/10 rounded-2xl bg-white/50 focus:outline-none focus:ring-2 focus:ring-black/20 sm:text-sm"
          />
        </div>
        <div>
          <label for="password" class="block text-sm font-bold text-gray-700 font-heading">PASSWORD</label>
          <input
            v-model="form.password"
            type="password"
            id="password"
            required
            class="mt-1 block w-full px-4 py-2.5 border border-black/10 rounded-2xl bg-white/50 focus:outline-none focus:ring-2 focus:ring-black/20 sm:text-sm"
          />
        </div>
        <div>
          <label for="re_password" class="block text-sm font-bold text-gray-700 font-heading">CONFIRM PASSWORD</label>
          <input
            v-model="form.re_password"
            type="password"
            id="re_password"
            required
            class="mt-1 block w-full px-4 py-2.5 border border-black/10 rounded-2xl bg-white/50 focus:outline-none focus:ring-2 focus:ring-black/20 sm:text-sm"
          />
        </div>
        <div v-if="errorMsg" class="text-red-600 text-sm">
          {{ errorMsg }}
        </div>
        <button
          type="submit"
          :disabled="loading"
          class="w-full bg-black text-white py-3 rounded-full text-sm font-bold shadow-md hover:bg-gray-800 transition-all active:scale-95 font-heading cursor-pointer disabled:opacity-50"
        >
          {{ loading ? 'CREATING ACCOUNT...' : 'SIGN UP' }}
        </button>
      </form>
      <div class="mt-6 text-center text-sm">
        <span class="text-gray-500">Already have an account?</span>
        <router-link to="/login" class="ml-1 font-bold text-black hover:underline font-heading">LOGIN</router-link>
      </div>
    </div>
  </template>

  <script setup lang="ts">
  import { ref } from 'vue';
  import { useRouter } from 'vue-router';
  import request from '../api/request';

  const form = ref({ username: '', password: '', re_password: '' });
  const loading = ref(false);
  const errorMsg = ref('');
  const router = useRouter();

  const handleSignup = async () => {
    if (form.value.password !== form.value.re_password) {
      errorMsg.value = 'Passwords do not match';
      return;
    }

    loading.value = true;
    errorMsg.value = '';
    try {
      const res: any = await request.post('/signup', form.value);
      if (res.code === 1000) {
        alert('Signup successful! Please login.');
        router.push('/login');
      } else {
        errorMsg.value = res.msg || 'Signup failed';
      }
    } catch (err: any) {
      errorMsg.value = err.response?.data?.msg || 'An error occurred';
    } finally {
      loading.value = false;
    }
  };
  </script>
  ```

- [ ] **Step 3: Run typecheck to verify auth components**
  Run: `npm --prefix frontend run typecheck`
  Expected: Pass.

- [ ] **Step 4: Commit**
  ```bash
  git add frontend/src/pages/Login.vue frontend/src/pages/Signup.vue
  git commit -m "feat: refactor Login and Signup pages to use glass forms and labels"
  ```
