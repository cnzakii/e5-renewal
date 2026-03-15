import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import { useAuth } from './stores/auth'
import './assets/main.css'

// Restore authentication state from storage
const { init } = useAuth()
init()

createApp(App).use(router).mount('#app')
