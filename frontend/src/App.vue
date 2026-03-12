<template>
  <div class="min-h-screen flex flex-col">
    <div class="bg-red-500 text-white p-2 text-center text-xs">Diagnostic: App Loaded</div>
    <Navbar />
    <main class="flex-grow max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 w-full">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import Navbar from './components/Navbar.vue';
</script>