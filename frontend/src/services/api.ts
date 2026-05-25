import axios from 'axios'
import { useAuthStore } from '@/store/authStore'

// Authenticated Axios instance — the JWT interceptor reads the current token
// from Zustand at call-time (not at creation-time) so it's always fresh.
const api = axios.create({
  baseURL: '/api',
  headers: { 'Content-Type': 'application/json' },
})

api.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      useAuthStore.getState().logout()
      window.location.replace('/login')
    }
    return Promise.reject(err)
  },
)

export default api
