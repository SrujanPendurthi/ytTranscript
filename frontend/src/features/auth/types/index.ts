export type Plan = 'free' | 'pro' | 'enterprise'

export interface User {
  id: string
  email: string
  plan: Plan
  is_active: boolean
  stripe_customer_id?: string
  created_at: string
  updated_at: string
}

export interface AuthResponse {
  token: string
  user: User
}

export interface LoginRequest {
  email: string
  password: string
}
