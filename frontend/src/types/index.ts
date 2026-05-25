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

export type JobStatus = 'pending' | 'processing' | 'completed' | 'failed'

export interface Segment {
  start: number
  end: number
  text: string
}

export interface Job {
  id: string
  user_id: string
  youtube_url: string
  status: JobStatus
  transcript_text?: string
  language?: string
  duration_seconds?: number
  whisper_used: boolean
  segments?: Segment[]
  error_message?: string
  created_at: string
  updated_at: string
}

export interface ApiKey {
  id: string
  name: string
  prefix: string
  last_used_at?: string
  created_at: string
  updated_at: string
}

export interface CreatedApiKey extends ApiKey {
  raw_key: string
}

export interface ApiError {
  error: string
}
