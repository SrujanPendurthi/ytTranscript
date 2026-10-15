import { createContext, useContext, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Sparkles, Youtube, ArrowRight, Loader2, ChevronDown, Play,
  Copy, Download, FileJson, Inbox, Zap, ArrowUpRight,
} from 'lucide-react'
import { useAuthStore } from '@/features/auth'
import api from '@/config/api'
import { cn } from '@/utils'
import type { Job, JobStatus, Segment, TranscriptSource, Usage } from '../types'

// ---------------------------------------------------------------------------
// Lightweight toast (avoids prop-drilling through the job actions)
// ---------------------------------------------------------------------------

const ToastContext = createContext<(msg: string) => void>(() => {})
const useToast = () => useContext(ToastContext)

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

const EXAMPLES = [
  { label: 'a talk', url: 'https://youtube.com/watch?v=dQw4w9WgXcQ' },
  { label: 'a lecture', url: 'https://youtu.be/M7lc1UVf-VE' },
]

export default function DashboardPage() {
  const [toast, setToast] = useState<string | null>(null)
  const notify = (msg: string) => {
    setToast(msg)
    window.clearTimeout((notify as { _t?: number })._t)
    ;(notify as { _t?: number })._t = window.setTimeout(() => setToast(null), 2200)
  }

  const { data: jobs = [], isFetching } = useQuery<Job[]>({
    queryKey: ['jobs'],
    queryFn: async () => (await api.get<Job[]>('/v1/jobs')).data,
    refetchInterval: (query) =>
      query.state.data?.some((j) => j.status === 'pending' || j.status === 'processing')
        ? 3000
        : false,
  })

  return (
    <ToastContext.Provider value={notify}>
      {/* PRIMARY — the funnel */}
      <Composer />

      {/* SECONDARY + TERTIARY */}
      <div className="mx-auto mt-4 grid max-w-5xl gap-7 lg:grid-cols-[1fr_296px]">
        <RecentTranscripts jobs={jobs} isFetching={isFetching} />
        <UsageRail jobs={jobs} />
      </div>

      {/* Toast */}
      <div
        aria-live="polite"
        className={cn(
          'fixed bottom-6 left-1/2 z-50 flex -translate-x-1/2 items-center gap-2 rounded-xl',
          'bg-foreground px-4 py-2.5 text-sm text-background shadow-lg transition-all duration-200',
          toast ? 'translate-y-0 opacity-100' : 'pointer-events-none translate-y-3 opacity-0',
        )}
      >
        {toast}
      </div>
    </ToastContext.Provider>
  )
}

// ---------------------------------------------------------------------------
// Composer (hero)
// ---------------------------------------------------------------------------

function Composer() {
  const [url, setUrl] = useState('')
  const queryClient = useQueryClient()

  const { mutate, isPending, error } = useMutation<Job, Error, string>({
    mutationFn: async (youtubeUrl) =>
      (await api.post<Job>('/v1/transcribe', { youtube_url: youtubeUrl })).data,
    onSuccess: () => {
      setUrl('')
      queryClient.invalidateQueries({ queryKey: ['jobs'] })
    },
  })

  const errorMsg =
    (error as { response?: { data?: { error?: string } } })?.response?.data?.error ??
    (error ? 'Something went wrong. Check the link and try again.' : null)

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    const value = url.trim() || EXAMPLES[0].url
    mutate(value)
  }

  return (
    <section className="px-1 pb-2 pt-6 text-center">
      <div className="mx-auto max-w-xl">
        <p className="inline-flex items-center gap-1.5 font-mono text-[11px] uppercase tracking-[0.15em] text-muted-foreground">
          <Sparkles className="h-3.5 w-3.5 text-violet-500" />
          New transcript
        </p>
        <h1 className="mt-3 text-balance text-4xl font-bold tracking-tight text-foreground">
          Transcribe any YouTube video
        </h1>
        <p className="mx-auto mt-3 max-w-md text-balance text-muted-foreground">
          Paste a link — captions come back in seconds. No captions? We transcribe the audio for you.
        </p>

        <form onSubmit={submit} className="mt-7 flex flex-col gap-2.5 sm:flex-row">
          <label className="flex flex-1 items-center rounded-2xl border border-input bg-card shadow-sm transition focus-within:border-primary focus-within:ring-4 focus-within:ring-primary/15">
            <Youtube className="ml-4 h-[18px] w-[18px] shrink-0 text-muted-foreground" />
            <input
              autoFocus
              type="text"
              inputMode="url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="Paste a YouTube URL…"
              className="w-full min-w-0 bg-transparent px-3.5 py-4 font-mono text-sm text-foreground outline-none placeholder:text-muted-foreground/70"
            />
          </label>
          <button
            type="submit"
            disabled={isPending}
            className={cn(
              'inline-flex shrink-0 items-center justify-center gap-2 rounded-2xl px-6 py-4 sm:py-0',
              'bg-gradient-to-br from-indigo-600 to-violet-500 text-[15px] font-semibold text-white',
              'shadow-[0_10px_22px_-8px_rgba(91,87,240,0.6)] transition hover:brightness-105',
              'active:translate-y-px disabled:opacity-60',
            )}
          >
            {isPending ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" /> Transcribing…
              </>
            ) : (
              <>
                Transcribe <ArrowRight className="h-4 w-4" />
              </>
            )}
          </button>
        </form>

        {/* friction reducer: one-click examples */}
        <div className="mt-4 flex flex-wrap items-center justify-center gap-2 text-[13px] text-muted-foreground">
          <span>Try:</span>
          {EXAMPLES.map((ex) => (
            <button
              key={ex.url}
              type="button"
              onClick={() => setUrl(ex.url)}
              className="rounded-full border border-border bg-card px-3 py-1 font-mono text-xs text-muted-foreground transition hover:border-primary/40 hover:text-primary"
            >
              {ex.label}
            </button>
          ))}
        </div>

        {errorMsg ? (
          <p className="mx-auto mt-4 max-w-md rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive">
            {errorMsg}
          </p>
        ) : (
          /* reassurance at the decision point */
          <div className="mt-6 flex flex-wrap items-center justify-center gap-x-3.5 gap-y-2 text-xs text-muted-foreground">
            <span className="inline-flex items-center gap-2">
              <span className="inline-flex items-center gap-1.5">
                <Dot className="bg-teal-500" />
                <b className="font-semibold text-foreground">Captions</b>
              </span>
              <span className="text-muted-foreground/60">→</span>
              <span className="inline-flex items-center gap-1.5">
                <Dot className="bg-violet-500" /> Whisper AI
              </span>
            </span>
            <Divider />
            <span>
              <b className="font-semibold text-foreground">71%</b> return instantly
            </span>
            <Divider />
            <span>
              avg <b className="font-semibold text-foreground">0:38</b>
            </span>
            {isPending && <Equalizer />}
          </div>
        )}
      </div>
    </section>
  )
}

// ---------------------------------------------------------------------------
// Recent transcripts (secondary)
// ---------------------------------------------------------------------------

type Filter = 'all' | 'completed' | 'failed'

function RecentTranscripts({ jobs, isFetching }: { jobs: Job[]; isFetching: boolean }) {
  const [filter, setFilter] = useState<Filter>('all')
  const shown = jobs.filter((j) => filter === 'all' || j.status === filter)

  return (
    <section>
      <div className="mb-3.5 flex items-baseline gap-3 px-0.5">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Recent</h3>
        <span className="font-mono text-xs text-muted-foreground/70">
          {jobs.length} transcript{jobs.length === 1 ? '' : 's'}
        </span>
        {isFetching && <Loader2 className="h-3 w-3 animate-spin text-muted-foreground/60" />}
        <div className="ml-auto flex gap-1.5">
          {(['all', 'completed', 'failed'] as Filter[]).map((f) => (
            <button
              key={f}
              onClick={() => setFilter(f)}
              className={cn(
                'rounded-full border px-2.5 py-1 text-xs capitalize transition',
                filter === f
                  ? 'border-primary/25 bg-accent text-accent-foreground'
                  : 'border-border bg-card text-muted-foreground hover:text-foreground',
              )}
            >
              {f}
            </button>
          ))}
        </div>
      </div>

      {jobs.length === 0 ? (
        <EmptyState fetching={isFetching} />
      ) : shown.length === 0 ? (
        <p className="rounded-xl border border-dashed border-border bg-card/50 px-4 py-8 text-center text-sm text-muted-foreground">
          No {filter} transcripts.
        </p>
      ) : (
        <div className="flex flex-col gap-2.5">
          {shown.map((job) => (
            <JobRow key={job.id} job={job} />
          ))}
        </div>
      )}
    </section>
  )
}

function EmptyState({ fetching }: { fetching: boolean }) {
  if (fetching) {
    return (
      <div className="flex items-center justify-center rounded-xl border border-border bg-card py-14 text-muted-foreground">
        <Loader2 className="h-5 w-5 animate-spin" />
      </div>
    )
  }
  return (
    <div className="rounded-xl border border-dashed border-border bg-card/60 px-6 py-12 text-center">
      <div className="mx-auto flex h-11 w-11 items-center justify-center rounded-xl bg-accent text-accent-foreground">
        <Inbox className="h-5 w-5" />
      </div>
      <p className="mt-3 text-sm font-medium text-foreground">No transcripts yet</p>
      <p className="mx-auto mt-1 max-w-xs text-sm text-muted-foreground">
        Paste a YouTube link above to create your first transcript.
      </p>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Job row
// ---------------------------------------------------------------------------

function JobRow({ job: initial }: { job: Job }) {
  const [open, setOpen] = useState(false)

  const { data: job = initial } = useQuery<Job>({
    queryKey: ['job', initial.id],
    queryFn: async () => (await api.get<Job>(`/v1/jobs/${initial.id}`)).data,
    initialData: initial,
    refetchInterval: (query) => {
      const s = query.state.data?.status
      return s === 'pending' || s === 'processing' ? 2000 : false
    },
  })

  const hasContent = Boolean(job.transcript_text || job.segments?.length)
  const vid = videoId(job.youtube_url)

  return (
    <article className="overflow-hidden rounded-xl border border-border bg-card shadow-sm transition hover:border-input">
      <button
        onClick={() => hasContent && setOpen((v) => !v)}
        aria-expanded={open}
        className={cn(
          'grid w-full grid-cols-[auto_1fr_auto] items-center gap-3.5 p-3.5 text-left sm:grid-cols-[auto_1fr_auto_auto]',
          hasContent ? 'cursor-pointer' : 'cursor-default',
        )}
      >
        <span className="grid h-11 w-11 place-items-center rounded-lg border border-border bg-gradient-to-br from-indigo-50 to-violet-50 text-muted-foreground">
          <Play className="h-[17px] w-[17px]" />
        </span>

        <span className="min-w-0">
          <span className="block truncate text-sm font-semibold text-foreground">{titleFor(job)}</span>
          <span className="mt-1 flex flex-wrap items-center gap-2 text-xs text-muted-foreground/80">
            <span className="font-mono text-muted-foreground">{vid}</span>
            {job.language && (
              <>
                <Sep />
                <span className="font-mono text-muted-foreground">{job.language.toUpperCase()}</span>
              </>
            )}
            {job.duration_seconds != null && (
              <>
                <Sep />
                <span className="font-mono text-muted-foreground">{formatDuration(job.duration_seconds)}</span>
              </>
            )}
          </span>
        </span>

        <span className="hidden sm:block">{job.source && <SourceChip source={job.source} />}</span>

        <span className="flex items-center gap-3">
          <StatusPill status={job.status} />
          {hasContent && (
            <ChevronDown className={cn('h-4 w-4 shrink-0 text-muted-foreground transition', open && 'rotate-180')} />
          )}
        </span>
      </button>

      {open && hasContent && (
        <div className="border-t border-border px-3.5 pb-3.5 pt-0.5">
          {job.error_message && (
            <p className="mt-3 rounded-lg border border-destructive/20 bg-destructive/[0.07] px-3 py-2.5 text-sm text-destructive">
              {job.error_message}
            </p>
          )}
          {job.segments?.length ? (
            <SegmentList segments={job.segments} />
          ) : job.transcript_text ? (
            <div className="mt-3 max-h-60 overflow-y-auto rounded-lg border border-border bg-muted/40 px-3.5 py-3 text-sm text-foreground/90">
              {job.transcript_text}
            </div>
          ) : null}
          <JobActions job={job} />
        </div>
      )}
    </article>
  )
}

function SegmentList({ segments }: { segments: Segment[] }) {
  return (
    <div className="mt-3 max-h-60 overflow-y-auto rounded-lg border border-border bg-muted/30">
      {segments.map((seg, i) => (
        <div
          key={i}
          className="grid grid-cols-[52px_1fr] gap-3 border-b border-border px-3 py-2 text-[13.5px] last:border-0 hover:bg-card"
        >
          <time className="font-mono text-xs tabular-nums text-primary">{formatTs(seg.start)}</time>
          <p className="m-0 text-foreground/85">{seg.text}</p>
        </div>
      ))}
    </div>
  )
}

function JobActions({ job }: { job: Job }) {
  const notify = useToast()
  const text = job.transcript_text ?? job.segments?.map((s) => s.text).join(' ') ?? ''

  const copy = async () => {
    await navigator.clipboard.writeText(text)
    notify('Copied to clipboard')
  }
  const downloadSrt = () => {
    if (!job.segments?.length) return
    downloadFile(`${videoId(job.youtube_url)}.srt`, toSrt(job.segments), 'text/plain')
    notify('Downloaded .srt')
  }
  const exportJson = () => {
    downloadFile(`${videoId(job.youtube_url)}.json`, JSON.stringify(job, null, 2), 'application/json')
    notify('Exported JSON')
  }

  return (
    <div className="mt-3 flex flex-wrap gap-2">
      {text && <ActionButton icon={Copy} label="Copy text" onClick={copy} />}
      {job.segments?.length ? <ActionButton icon={Download} label="Download .srt" onClick={downloadSrt} /> : null}
      <ActionButton icon={FileJson} label="Export JSON" onClick={exportJson} />
    </div>
  )
}

function ActionButton({
  icon: Icon,
  label,
  onClick,
}: {
  icon: React.ElementType
  label: string
  onClick: () => void
}) {
  return (
    <button
      onClick={onClick}
      className="inline-flex items-center gap-1.5 rounded-lg border border-input bg-card px-3 py-1.5 text-xs text-muted-foreground transition hover:border-primary/40 hover:text-primary"
    >
      <Icon className="h-3.5 w-3.5" />
      {label}
    </button>
  )
}

// ---------------------------------------------------------------------------
// Usage rail (tertiary)
// ---------------------------------------------------------------------------

function UsageRail({ jobs }: { jobs: Job[] }) {
  const user = useAuthStore((s) => s.user)

  const { data: usage } = useQuery<Usage>({
    queryKey: ['usage'],
    queryFn: async () => (await api.get<Usage>('/v1/usage')).data,
    refetchInterval: 15000,
  })

  const hitRate = useMemo(() => {
    const completed = jobs.filter((j) => j.status === 'completed')
    const instant = completed.filter((j) => !j.whisper_used).length
    return completed.length ? Math.round((instant / completed.length) * 100) : 0
  }, [jobs])

  const used = usage?.minutes_used ?? 0
  const included = usage?.included_minutes ?? 0
  const overage = usage?.overage_minutes ?? 0
  const pct = included ? Math.min(100, Math.round((used / included) * 100)) : 0
  const nearCap = pct >= 80

  return (
    <aside className="flex flex-col gap-3.5 lg:sticky lg:top-[76px]">
      <div className="rounded-xl border border-border bg-card p-4 shadow-sm">
        <h4 className="mb-3.5 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
          This period
        </h4>

        <div className="mb-1.5 flex items-baseline justify-between">
          <span className="text-[13px] text-muted-foreground">Minutes used</span>
          <span className="font-mono text-sm tabular-nums">
            {used.toLocaleString()} <span className="text-muted-foreground/70">/ {included.toLocaleString()}</span>
          </span>
        </div>
        <div className="h-1.5 overflow-hidden rounded bg-muted">
          <div
            className={cn(
              'h-full rounded transition-all',
              nearCap ? 'bg-gradient-to-r from-amber-500 to-red-500' : 'bg-gradient-to-r from-indigo-600 to-violet-500',
            )}
            style={{ width: `${pct}%` }}
          />
        </div>
        {overage > 0 ? (
          <p className="mt-2 text-[11.5px] text-amber-600">+{overage.toLocaleString()} overage min this period</p>
        ) : (
          <p className="mt-2 text-[11.5px] text-muted-foreground/80">{usage?.jobs_count ?? 0} transcripts this period</p>
        )}

        <div className="mt-4 border-t border-border pt-3.5">
          <div className="mb-1.5 flex items-baseline justify-between">
            <span className="text-[13px] text-muted-foreground">Instant hit rate</span>
            <span className="font-mono text-sm tabular-nums">{hitRate}%</span>
          </div>
          <div className="h-1.5 overflow-hidden rounded bg-muted">
            <div className="h-full rounded bg-teal-500 transition-all" style={{ width: `${hitRate}%` }} />
          </div>
          <p className="mt-2 text-[11.5px] text-muted-foreground/80">Returned from captions, no audio pass needed.</p>
        </div>
      </div>

      {user?.plan === 'free' && <UpgradeCard />}
    </aside>
  )
}

function UpgradeCard() {
  const notify = useToast()
  const { mutate, isPending } = useMutation<{ checkout_url: string }, Error>({
    mutationFn: async () => (await api.post<{ checkout_url: string }>('/billing/checkout')).data,
    onSuccess: ({ checkout_url }) => {
      if (checkout_url) window.location.href = checkout_url
    },
    onError: () => notify('Could not start checkout'),
  })

  return (
    <div className="rounded-xl border border-violet-200/70 bg-gradient-to-br from-violet-50 to-card p-4 shadow-sm">
      <p className="text-sm font-semibold text-foreground">Go unlimited</p>
      <p className="mt-1 text-[12.5px] text-muted-foreground">
        Remove the monthly cap, add API keys, and priority processing.
      </p>
      <button
        onClick={() => mutate()}
        disabled={isPending}
        className="mt-3 inline-flex w-full items-center justify-center gap-2 rounded-lg bg-gradient-to-br from-indigo-600 to-violet-500 py-2.5 text-sm font-semibold text-white shadow-sm transition hover:brightness-105 disabled:opacity-60"
      >
        {isPending ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : (
          <>
            <Zap className="h-3.5 w-3.5" /> Upgrade to Pro — $10/mo <ArrowUpRight className="h-3.5 w-3.5" />
          </>
        )}
      </button>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Small shared pieces
// ---------------------------------------------------------------------------

function Dot({ className }: { className?: string }) {
  return <span className={cn('inline-block h-[7px] w-[7px] rounded-full', className)} />
}
function Sep() {
  return <span className="opacity-50">·</span>
}
function Divider() {
  return <span className="h-3 w-px bg-border" />
}

function Equalizer() {
  return (
    <span className="ml-1 flex h-3 items-end gap-[2px]" aria-hidden>
      {[0, 1, 2, 3, 4].map((i) => (
        <span
          key={i}
          className="w-[2.5px] rounded-full bg-violet-500 motion-reduce:animate-none animate-eq"
          style={{ height: '30%', animationDelay: `${i * 150}ms` }}
        />
      ))}
    </span>
  )
}

const statusStyles: Record<JobStatus, { label: string; text: string; bg: string; dot: string; pulse?: boolean }> = {
  pending: { label: 'Queued', text: 'text-slate-600', bg: 'bg-slate-100', dot: 'bg-slate-400' },
  processing: { label: 'Processing', text: 'text-amber-700', bg: 'bg-amber-50', dot: 'bg-amber-500', pulse: true },
  completed: { label: 'Completed', text: 'text-emerald-700', bg: 'bg-emerald-50', dot: 'bg-emerald-500' },
  failed: { label: 'Failed', text: 'text-red-700', bg: 'bg-red-50', dot: 'bg-red-500' },
}

function StatusPill({ status }: { status: JobStatus }) {
  const s = statusStyles[status]
  return (
    <span className={cn('inline-flex items-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-semibold', s.text, s.bg)}>
      <span className={cn('h-1.5 w-1.5 rounded-full', s.dot, s.pulse && 'animate-pulse')} />
      {s.label}
    </span>
  )
}

const sourceStyles: Record<TranscriptSource, { label: string; text: string; dot: string }> = {
  cache: { label: 'cache', text: 'text-slate-500', dot: 'bg-slate-400' },
  captions: { label: 'captions', text: 'text-teal-600', dot: 'bg-teal-500' },
  whisper: { label: 'whisper', text: 'text-violet-600', dot: 'bg-violet-500' },
}

function SourceChip({ source }: { source: TranscriptSource }) {
  const s = sourceStyles[source]
  return (
    <span className={cn('inline-flex items-center gap-1.5 font-mono text-[11px]', s.text)}>
      <span className={cn('h-1.5 w-1.5 rounded-full', s.dot)} />
      {s.label}
    </span>
  )
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function videoId(url: string): string {
  try {
    const u = new URL(url)
    if (u.hostname === 'youtu.be') return u.pathname.slice(1) || url
    const v = u.searchParams.get('v')
    if (v) return v
    const m = u.pathname.match(/\/(shorts|embed)\/([\w-]+)/)
    if (m) return m[2]
  } catch {
    /* not a URL — fall through */
  }
  return url
}

function titleFor(job: Job): string {
  // The API stores only the URL; surface the id as a stable, readable title.
  const id = videoId(job.youtube_url)
  return id === job.youtube_url ? job.youtube_url : `youtube.com/${id}`
}

function formatDuration(s: number): string {
  const total = Math.round(s)
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const sec = total % 60
  const mm = h ? String(m).padStart(2, '0') : String(m)
  return `${h ? `${h}:` : ''}${mm}:${String(sec).padStart(2, '0')}`
}

function formatTs(s: number): string {
  const m = Math.floor(s / 60)
  return `${m}:${(s % 60).toFixed(1).padStart(4, '0')}`
}

function srtTime(s: number): string {
  const ms = Math.round(s * 1000)
  const h = Math.floor(ms / 3_600_000)
  const m = Math.floor((ms % 3_600_000) / 60_000)
  const sec = Math.floor((ms % 60_000) / 1000)
  const milli = ms % 1000
  const p = (n: number, w = 2) => String(n).padStart(w, '0')
  return `${p(h)}:${p(m)}:${p(sec)},${p(milli, 3)}`
}

function toSrt(segments: Segment[]): string {
  return segments
    .map((seg, i) => `${i + 1}\n${srtTime(seg.start)} --> ${srtTime(seg.end)}\n${seg.text.trim()}\n`)
    .join('\n')
}

function downloadFile(name: string, content: string, type: string) {
  const blob = new Blob([content], { type })
  const href = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = href
  a.download = name
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(href)
}
