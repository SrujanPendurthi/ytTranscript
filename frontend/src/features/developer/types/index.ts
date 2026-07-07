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
