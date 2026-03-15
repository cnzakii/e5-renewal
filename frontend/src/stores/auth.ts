import { ref, computed } from 'vue'

const token = ref('')
const isAuthenticated = computed(() => token.value !== '')

function setAuth(newToken: string, remember: boolean) {
  token.value = newToken
  localStorage.removeItem('token')
  sessionStorage.removeItem('token')
  const storage = remember ? localStorage : sessionStorage
  storage.setItem('token', newToken)
}

function clearAuth() {
  token.value = ''
  localStorage.removeItem('token')
  sessionStorage.removeItem('token')
}

function init() {
  const savedToken = localStorage.getItem('token') || sessionStorage.getItem('token')
  if (savedToken) {
    token.value = savedToken
  }
}

export function useAuth() {
  return { token, isAuthenticated, setAuth, clearAuth, init }
}
