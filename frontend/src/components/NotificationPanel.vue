<template>
  <div v-if="store.connected" class="fixed bottom-4 right-4 z-50 max-w-sm">
    <div class="bg-white rounded-lg shadow-xl border border-gray-200 overflow-hidden">
      <div class="bg-indigo-600 px-4 py-2 flex justify-between items-center">
        <span class="text-white font-medium">通知 ({{ store.notifications.length }})</span>
        <div class="flex items-center space-x-2">
          <span class="text-xs text-indigo-200">{{ store.onlineUsers }} 在线</span>
          <span class="w-2 h-2 bg-green-400 rounded-full"></span>
          <button @click="toggleExpanded" class="text-white hover:text-indigo-200">
            <svg v-if="expanded" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
            </svg>
            <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7"></path>
            </svg>
          </button>
          <button @click="clearAll" class="text-white hover:text-indigo-200 text-xs">清除</button>
        </div>
      </div>

      <div v-if="expanded" class="max-h-96 overflow-y-auto">
        <div v-if="store.notifications.length === 0" class="px-4 py-8 text-center text-gray-500">
          暂无新通知
        </div>
        <div
          v-for="(notification, index) in store.notifications"
          :key="notification.timestamp"
          class="px-4 py-3 border-b border-gray-100 hover:bg-gray-50 transition-colors"
          :class="getNotificationClass(notification.type)"
        >
          <div class="flex justify-between items-start">
            <div class="flex-1">
              <p class="font-medium text-gray-900">{{ notification.title }}</p>
              <p class="text-sm text-gray-600 mt-1">{{ notification.content }}</p>
              <p v-if="notification.timestamp" class="text-xs text-gray-400 mt-2">
                {{ formatTime(notification.timestamp) }}
              </p>
            </div>
            <button @click="dismiss(index)" class="text-gray-400 hover:text-gray-600 ml-2">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
              </svg>
            </button>
          </div>
        </div>
      </div>

      <div v-else class="px-4 py-2 text-sm text-gray-500">
        <span v-if="store.notifications.length > 0">
          {{ store.notifications[0]?.title }}: {{ store.notifications[0]?.content }}
        </span>
        <span v-else>已连接，等待通知...</span>
      </div>
    </div>
  </div>

  <div v-else-if="showConnectionStatus" class="fixed bottom-4 right-4 z-50">
    <div class="bg-gray-800 text-white px-4 py-2 rounded-lg shadow-lg flex items-center space-x-2">
      <span class="w-2 h-2 bg-red-500 rounded-full animate-pulse"></span>
      <span class="text-sm">WebSocket 连接断开，正在重连...</span>
    </div>
  </div>
</template>

<script setup lang="ts">import { ref, onMounted, onUnmounted, watch } from 'vue';
import { useWebSocketStore, wsService } from '../store/websocket';
import { useAuthStore } from '../store/auth';
const store = useWebSocketStore();
const authStore = useAuthStore();
const expanded = ref(false);
const showConnectionStatus = ref(false);
let connectionCheckInterval: number | null = null;
const toggleExpanded = () => {
  expanded.value = !expanded.value;
};
const clearAll = () => {
  store.clearNotifications();
};
const dismiss = (index: number) => {
  store.removeNotification(index);
};
const getNotificationClass = (type: string): string => {
  switch (type) {
    case 'comment':
      return 'border-l-4 border-indigo-500';
    case 'vote':
      return 'border-l-4 border-green-500';
    case 'follow':
      return 'border-l-4 border-blue-500';
    case 'mention':
      return 'border-l-4 border-purple-500';
    case 'system':
      return 'border-l-4 border-gray-500';
    default:
      return 'border-l-4 border-gray-300';
  }
};
const formatTime = (timestamp: number): string => {
  const date = new Date(timestamp);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  if (diff < 60000) {
    return '刚刚';
  } else if (diff < 3600000) {
    return `${Math.floor(diff / 60000)} 分钟前`;
  } else if (diff < 86400000) {
    return `${Math.floor(diff / 3600000)} 小时前`;
  } else {
    return date.toLocaleDateString();
  }
};
watch(() => store.connected, (connected) => {
  if (!connected && authStore.token) {
    showConnectionStatus.value = true;
    setTimeout(() => {
      showConnectionStatus.value = false;
    }, 5000);
  } else {
    showConnectionStatus.value = false;
  }
});
onMounted(() => {
  if (authStore.token) {
    wsService.initialize('ws://localhost:8081/ws', authStore.token, store);
    wsService.connect();
  }
  connectionCheckInterval = window.setInterval(() => {
    if (authStore.token && !wsService.isConnected() && wsService.reconnectAttempts < 5) {
      wsService.connect();
    }
  }, 10000);
});
onUnmounted(() => {
  wsService.disconnect();
  if (connectionCheckInterval !== null) {
    clearInterval(connectionCheckInterval);
  }
});</script>