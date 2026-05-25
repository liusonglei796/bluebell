<template>
  <div class="flex items-center justify-center min-h-[70vh]">
    <div class="glass rounded-3xl p-10 w-full max-w-md">
      <h2 class="text-3xl font-black text-center text-black tracking-tighter mb-8 font-heading">SIGN UP</h2>
      <form @submit.prevent="handleSignup" class="space-y-5">
        <div>
          <label for="username" class="block text-xs font-bold uppercase tracking-widest text-gray-500 mb-2">Username</label>
          <input v-model="form.username" type="text" id="username" required class="block w-full px-4 py-3 bg-white/50 border border-black/10 rounded-xl focus:outline-none focus:ring-2 focus:ring-black/20 backdrop-blur-sm text-sm" />
        </div>
        <div>
          <label for="password" class="block text-xs font-bold uppercase tracking-widest text-gray-500 mb-2">Password</label>
          <input v-model="form.password" type="password" id="password" required class="block w-full px-4 py-3 bg-white/50 border border-black/10 rounded-xl focus:outline-none focus:ring-2 focus:ring-black/20 backdrop-blur-sm text-sm" />
        </div>
        <div>
          <label for="re_password" class="block text-xs font-bold uppercase tracking-widest text-gray-500 mb-2">Confirm Password</label>
          <input v-model="form.re_password" type="password" id="re_password" required class="block w-full px-4 py-3 bg-white/50 border border-black/10 rounded-xl focus:outline-none focus:ring-2 focus:ring-black/20 backdrop-blur-sm text-sm" />
        </div>
        <div v-if="errorMsg" class="text-red-500 font-medium text-sm">
          {{ errorMsg }}
        </div>
        <button type="submit" :disabled="loading" class="w-full py-4 rounded-2xl bg-black text-white font-black text-sm uppercase tracking-wider hover:bg-gray-800 transition-all active:scale-95 shadow-lg disabled:opacity-50">
          {{ loading ? 'Signing up...' : 'Sign up' }}
        </button>
      </form>
      <div class="mt-6 text-center text-sm text-gray-500">
        Already have an account? <router-link to="/login" class="font-bold text-black hover:underline">Log in</router-link>
      </div>
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