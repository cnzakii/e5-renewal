import axios from 'axios'
import { useAuth } from '../stores/auth'
import { pathPrefix } from '../config'

export const apiClient = axios.create({
  baseURL: `${pathPrefix}/api`,
})

apiClient.interceptors.request.use((config) => {
  const { token } = useAuth()
  if (token.value) {
    config.headers['Authorization'] = `Bearer ${token.value}`
  }
  return config
})

apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      const { clearAuth } = useAuth()
      clearAuth()
      window.location.href = `${pathPrefix}/login`
    }
    return Promise.reject(error)
  }
)
