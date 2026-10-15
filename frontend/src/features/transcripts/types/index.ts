export type JobStatus = 'pending' | 'processing' | 'completed' | 'failed'

export type TranscriptSource = 'cache' | 'captions' | 'whisper'

export interface Segment {
  start: number
  end: number
  text: string
}

export interface Usage {
  plan: string
  included_minutes: number
  minutes_used: number
  overage_minutes: number
  jobs_count: number
  allow_overage: boolean
  period_start: string
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
  source?: TranscriptSource
  segments?: Segment[]
  error_message?: string
  created_at: string
  updated_at: string
}
