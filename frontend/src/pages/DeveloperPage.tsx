import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Copy, Check, Trash2, Plus, Eye, EyeOff, Loader2, Key } from 'lucide-react'
import api from '@/services/api'
import { cn } from '@/lib/utils'
import type { ApiKey, CreatedApiKey } from '@/types'

export default function DeveloperPage() {
  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-bold text-foreground">Developer Hub</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Manage API keys and integrate TranscriptFlow into your apps.
        </p>
      </div>

      <ApiKeysSection />
      <CurlExamples />
    </div>
  )
}

// ---------------------------------------------------------------------------
// API Keys
// ---------------------------------------------------------------------------

function ApiKeysSection() {
  const [newKeyName, setNewKeyName] = useState('')
  const [revealed, setRevealed] = useState<CreatedApiKey | null>(null)
  const queryClient = useQueryClient()

  const { data: keys = [], isLoading } = useQuery<ApiKey[]>({
    queryKey: ['api-keys'],
    queryFn: async () => {
      const res = await api.get<ApiKey[]>('/v1/keys')
      return res.data
    },
  })

  const createMutation = useMutation<CreatedApiKey, Error, string>({
    mutationFn: async (name) => {
      const res = await api.post<CreatedApiKey>('/v1/keys', { name })
      return res.data
    },
    onSuccess: (data) => {
      setNewKeyName('')
      setRevealed(data)
      queryClient.invalidateQueries({ queryKey: ['api-keys'] })
    },
  })

  const deleteMutation = useMutation<void, Error, string>({
    mutationFn: async (id) => {
      await api.delete(`/v1/keys/${id}`)
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['api-keys'] }),
  })

  return (
    <div className="rounded-lg border border-border bg-card shadow-sm">
      <div className="border-b border-border px-5 py-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">API Keys</h2>
      </div>

      {/* New key revealed banner */}
      {revealed && (
        <RevealedKeyBanner key={revealed.id} created={revealed} onDismiss={() => setRevealed(null)} />
      )}

      {/* Create form */}
      <div className="px-5 py-4 border-b border-border">
        <form
          onSubmit={(e) => { e.preventDefault(); if (newKeyName.trim()) createMutation.mutate(newKeyName.trim()) }}
          className="flex gap-2"
        >
          <input
            type="text"
            value={newKeyName}
            onChange={(e) => setNewKeyName(e.target.value)}
            placeholder="Key name (e.g. production)"
            className={cn(
              'flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm',
              'text-foreground placeholder:text-muted-foreground',
              'focus:outline-none focus:ring-2 focus:ring-ring',
            )}
          />
          <button
            type="submit"
            disabled={!newKeyName.trim() || createMutation.isPending}
            className="flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground transition-opacity hover:opacity-90 disabled:opacity-50"
          >
            {createMutation.isPending
              ? <Loader2 className="h-4 w-4 animate-spin" />
              : <Plus className="h-4 w-4" />}
            Create
          </button>
        </form>
      </div>

      {/* Key list */}
      <div className="divide-y divide-border">
        {isLoading && (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        )}

        {!isLoading && keys.length === 0 && (
          <div className="flex flex-col items-center gap-2 py-10 text-center">
            <Key className="h-8 w-8 text-muted-foreground/40" />
            <p className="text-sm text-muted-foreground">No API keys yet. Create one above.</p>
          </div>
        )}

        {keys.map((key) => (
          <div key={key.id} className="flex items-center gap-3 px-5 py-3">
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium text-foreground">{key.name}</p>
              <p className="mt-0.5 font-mono text-xs text-muted-foreground">
                {key.prefix}••••••••••••••••••••••••••••••••••••••••••••••••••
              </p>
            </div>
            <div className="shrink-0 text-right">
              <p className="text-xs text-muted-foreground">
                Created {new Date(key.created_at).toLocaleDateString()}
              </p>
              {key.last_used_at && (
                <p className="text-xs text-muted-foreground">
                  Last used {new Date(key.last_used_at).toLocaleDateString()}
                </p>
              )}
            </div>
            <button
              onClick={() => deleteMutation.mutate(key.id)}
              disabled={deleteMutation.isPending}
              aria-label="Revoke key"
              className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive disabled:opacity-50"
            >
              <Trash2 className="h-4 w-4" />
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}

function RevealedKeyBanner({ created, onDismiss }: { created: CreatedApiKey; onDismiss: () => void }) {
  const [visible, setVisible] = useState(false)
  const [copied, setCopied] = useState(false)

  const copy = async () => {
    await navigator.clipboard.writeText(created.raw_key)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="border-b border-border bg-primary/5 px-5 py-4">
      <p className="mb-2 text-sm font-semibold text-foreground">
        Key created — copy it now. It won't be shown again.
      </p>
      <div className="flex items-center gap-2">
        <div className="relative flex-1 overflow-hidden rounded-md border border-input bg-background px-3 py-2 font-mono text-sm text-foreground">
          {visible ? created.raw_key : '•'.repeat(created.raw_key.length)}
        </div>
        <button
          onClick={() => setVisible((v) => !v)}
          aria-label={visible ? 'Hide key' : 'Show key'}
          className="rounded-md border border-border p-2 text-muted-foreground transition-colors hover:text-foreground"
        >
          {visible ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
        </button>
        <button
          onClick={copy}
          className="rounded-md border border-border p-2 text-muted-foreground transition-colors hover:text-foreground"
        >
          {copied ? <Check className="h-4 w-4 text-primary" /> : <Copy className="h-4 w-4" />}
        </button>
      </div>
      <button
        onClick={onDismiss}
        className="mt-2 text-xs text-muted-foreground underline-offset-4 hover:underline"
      >
        I've copied it, dismiss
      </button>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Curl examples
// ---------------------------------------------------------------------------

const EXAMPLES = [
  {
    title: 'Submit a transcription job',
    code: `curl -X POST https://api.transcriptflow.io/api/v1/transcribe \\
  -H "X-API-Key: <your-key>" \\
  -H "Content-Type: application/json" \\
  -d '{"youtube_url":"https://youtu.be/dQw4w9WgXcQ"}'`,
  },
  {
    title: 'Poll job status',
    code: `curl https://api.transcriptflow.io/api/v1/jobs/<job-id> \\
  -H "X-API-Key: <your-key>"`,
  },
]

function CurlExamples() {
  return (
    <div className="space-y-4">
      <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
        API Examples
      </h2>
      {EXAMPLES.map((ex) => (
        <CodeBlock key={ex.title} title={ex.title} code={ex.code} />
      ))}
    </div>
  )
}

function CodeBlock({ title, code }: { title: string; code: string }) {
  const [copied, setCopied] = useState(false)

  const copy = async () => {
    await navigator.clipboard.writeText(code)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="rounded-lg border border-border bg-card shadow-sm">
      <div className="flex items-center justify-between border-b border-border px-4 py-2">
        <span className="text-xs font-medium text-muted-foreground">{title}</span>
        <button
          onClick={copy}
          className="flex items-center gap-1 rounded px-2 py-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
        >
          {copied ? <Check className="h-3 w-3 text-primary" /> : <Copy className="h-3 w-3" />}
          {copied ? 'Copied' : 'Copy'}
        </button>
      </div>
      <pre className="overflow-x-auto p-4 text-xs text-muted-foreground">{code}</pre>
    </div>
  )
}
