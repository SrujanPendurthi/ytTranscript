import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Copy, Check, Trash2, Plus, Eye, EyeOff, Loader2, KeyRound, Terminal } from 'lucide-react'
import api from '@/config/api'
import { cn } from '@/utils'
import type { ApiKey, CreatedApiKey } from '../types'

export default function DeveloperPage() {
  return (
    <div className="mx-auto max-w-3xl space-y-8">
      <div>
        <p className="font-mono text-[11px] uppercase tracking-[0.15em] text-muted-foreground">Developer</p>
        <h1 className="mt-2 text-2xl font-bold tracking-tight text-foreground">API keys &amp; integration</h1>
        <p className="mt-1.5 text-sm text-muted-foreground">
          Create a key and call TranscriptFlow from your own apps.
        </p>
      </div>

      <CurlExamples />
      <ApiKeysSection />
    </div>
  )
}

// ---------------------------------------------------------------------------
// API keys
// ---------------------------------------------------------------------------

function ApiKeysSection() {
  const [newKeyName, setNewKeyName] = useState('')
  const [revealed, setRevealed] = useState<CreatedApiKey | null>(null)
  const queryClient = useQueryClient()

  const { data: keys = [], isLoading } = useQuery<ApiKey[]>({
    queryKey: ['api-keys'],
    queryFn: async () => (await api.get<ApiKey[]>('/v1/keys')).data,
  })

  const createMutation = useMutation<CreatedApiKey, Error, string>({
    mutationFn: async (name) => (await api.post<CreatedApiKey>('/v1/keys', { name })).data,
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
    <div className="overflow-hidden rounded-xl border border-border bg-card shadow-sm">
      <div className="flex items-center justify-between border-b border-border px-5 py-3.5">
        <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">API keys</h2>
        {keys.length > 0 && (
          <span className="font-mono text-xs text-muted-foreground/70">
            {keys.length} active
          </span>
        )}
      </div>

      {revealed && (
        <RevealedKeyBanner key={revealed.id} created={revealed} onDismiss={() => setRevealed(null)} />
      )}

      {/* Create */}
      <div className="border-b border-border px-5 py-4">
        <form
          onSubmit={(e) => {
            e.preventDefault()
            if (newKeyName.trim()) createMutation.mutate(newKeyName.trim())
          }}
          className="flex gap-2"
        >
          <input
            type="text"
            value={newKeyName}
            onChange={(e) => setNewKeyName(e.target.value)}
            placeholder="Key name (e.g. production)"
            className="flex-1 rounded-lg border border-input bg-background px-3.5 py-2.5 text-sm text-foreground outline-none transition placeholder:text-muted-foreground/70 focus:border-primary focus:ring-4 focus:ring-primary/15"
          />
          <button
            type="submit"
            disabled={!newKeyName.trim() || createMutation.isPending}
            className="inline-flex shrink-0 items-center gap-1.5 rounded-lg bg-gradient-to-br from-indigo-600 to-violet-500 px-4 py-2.5 text-sm font-semibold text-white shadow-[0_10px_22px_-8px_rgba(91,87,240,0.6)] transition hover:brightness-105 active:translate-y-px disabled:opacity-60"
          >
            {createMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
            Create
          </button>
        </form>
      </div>

      {/* List */}
      <div className="divide-y divide-border">
        {isLoading && (
          <div className="flex items-center justify-center py-10">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        )}

        {!isLoading && keys.length === 0 && (
          <div className="flex flex-col items-center gap-3 py-12 text-center">
            <div className="grid h-11 w-11 place-items-center rounded-xl bg-accent text-accent-foreground">
              <KeyRound className="h-5 w-5" />
            </div>
            <p className="text-sm text-muted-foreground">No API keys yet — create one above.</p>
          </div>
        )}

        {keys.map((key) => (
          <div key={key.id} className="flex items-center gap-3 px-5 py-3.5">
            <div className="min-w-0 flex-1">
              <p className="text-sm font-semibold text-foreground">{key.name}</p>
              <p className="mt-0.5 truncate font-mono text-xs text-muted-foreground">
                {key.prefix}
                {'•'.repeat(28)}
              </p>
            </div>
            <div className="shrink-0 text-right text-xs text-muted-foreground">
              <p>Created {new Date(key.created_at).toLocaleDateString()}</p>
              {key.last_used_at && <p>Used {new Date(key.last_used_at).toLocaleDateString()}</p>}
            </div>
            <button
              onClick={() => deleteMutation.mutate(key.id)}
              disabled={deleteMutation.isPending}
              aria-label={`Revoke ${key.name}`}
              className="rounded-lg p-2 text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive disabled:opacity-50"
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
    <div className="border-b border-primary/15 bg-accent/60 px-5 py-4">
      <p className="mb-2 text-sm font-semibold text-foreground">
        Key created — copy it now. You won't be able to see it again.
      </p>
      <div className="flex items-center gap-2">
        <code className="flex-1 truncate rounded-lg border border-input bg-card px-3 py-2.5 font-mono text-sm text-foreground">
          {visible ? created.raw_key : '•'.repeat(Math.min(created.raw_key.length, 44))}
        </code>
        <button
          onClick={() => setVisible((v) => !v)}
          aria-label={visible ? 'Hide key' : 'Show key'}
          className="rounded-lg border border-input bg-card p-2.5 text-muted-foreground transition-colors hover:text-foreground"
        >
          {visible ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
        </button>
        <button
          onClick={copy}
          aria-label="Copy key"
          className="rounded-lg border border-input bg-card p-2.5 text-muted-foreground transition-colors hover:text-foreground"
        >
          {copied ? <Check className="h-4 w-4 text-emerald-600" /> : <Copy className="h-4 w-4" />}
        </button>
      </div>
      <button onClick={onDismiss} className="mt-2 text-xs text-muted-foreground underline-offset-4 hover:underline">
        I've copied it — dismiss
      </button>
    </div>
  )
}

// ---------------------------------------------------------------------------
// API examples
// ---------------------------------------------------------------------------

/** API root shown in the examples. Override with VITE_API_BASE_URL (e.g. a
 *  separate prod API host); otherwise use this origin + /api, which is correct
 *  in dev (Vite proxy) and prod (nginx proxy) alike. */
function apiBase(): string {
  const override = import.meta.env.VITE_API_BASE_URL?.replace(/\/+$/, '')
  return override || `${window.location.origin}/api`
}

function CurlExamples() {
  const base = apiBase()
  const examples = [
    {
      title: 'Submit a transcription job',
      code: `curl -X POST ${base}/v1/transcribe \\
  -H "X-API-Key: <your-key>" \\
  -H "Content-Type: application/json" \\
  -d '{"youtube_url":"https://youtu.be/dQw4w9WgXcQ"}'`,
    },
    {
      title: 'Poll job status',
      code: `curl ${base}/v1/jobs/<job-id> \\
  -H "X-API-Key: <your-key>"`,
    },
  ]

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2 px-0.5">
        <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Quickstart</h2>
        <BaseUrlChip base={base} />
      </div>
      {examples.map((ex) => (
        <CodeBlock key={ex.title} title={ex.title} code={ex.code} />
      ))}
    </div>
  )
}

function BaseUrlChip({ base }: { base: string }) {
  const [copied, setCopied] = useState(false)
  const copy = async () => {
    await navigator.clipboard.writeText(base)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }
  return (
    <button
      onClick={copy}
      title="Copy base URL"
      className="inline-flex items-center gap-1.5 rounded-lg border border-border bg-card px-2.5 py-1 font-mono text-xs text-muted-foreground transition hover:text-foreground"
    >
      {copied ? <Check className="h-3 w-3 text-primary" /> : <Copy className="h-3 w-3" />}
      {base}
    </button>
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
    <div className="overflow-hidden rounded-xl border border-border shadow-sm">
      <div className="flex items-center justify-between border-b border-border bg-muted/50 px-4 py-2.5">
        <span className="inline-flex items-center gap-2 font-mono text-xs text-muted-foreground">
          <Terminal className="h-3.5 w-3.5" />
          {title}
        </span>
        <button
          onClick={copy}
          className={cn(
            'inline-flex items-center gap-1.5 rounded px-2 py-1 text-xs transition-colors',
            copied ? 'text-primary' : 'text-muted-foreground hover:text-foreground',
          )}
        >
          {copied ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
          {copied ? 'Copied' : 'Copy'}
        </button>
      </div>
      <pre className="overflow-x-auto bg-card p-4 text-xs leading-relaxed text-foreground/85">{code}</pre>
    </div>
  )
}
