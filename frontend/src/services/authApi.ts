// Auth calls bypass the authenticated `api` instance and use raw axios so
// pre-login requests never trigger the 401 → logout redirect.
import axios from 'axios'
import type { AuthResponse, LoginRequest } from '@/types'

const raw = axios.create({
  baseURL: '/api',
  headers: { 'Content-Type': 'application/json' },
})

export async function login(data: LoginRequest): Promise<AuthResponse> {
  const res = await raw.post<AuthResponse>('/auth/login', data)
  return res.data
}

export async function signup(data: LoginRequest): Promise<AuthResponse> {
  const res = await raw.post<AuthResponse>('/auth/signup', data)
  return res.data
}
