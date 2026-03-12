import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './style.css'
import App from './App.vue'
import router from './router'
import { useAuthStore } from './store/auth'

const app = createApp(App)
const pinia = createPinia()

app.config.errorHandler = (err, _vm, info) => {
  console.error('Vue Error:', err, info)
}

app.use(pinia)
app.use(router)

// Initialize auth state from localStorage
try {
  const authStore = useAuthStore()
  authStore.init()
} catch (e) {
  console.error('Auth Store Init Error:', e)
}

app.mount('#app')