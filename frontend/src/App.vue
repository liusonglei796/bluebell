<template>
  <div class="min-h-screen flex flex-col">
    <Navbar />
    <main class="flex-grow max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-24 pb-8 w-full">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import Navbar from './components/Navbar.vue';
</script>