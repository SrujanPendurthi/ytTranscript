import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Loader2, CheckCircle2, XCircle, Clock, Zap, ArrowUpRight,
  Youtube, ChevronDown, ChevronUp,
} from 'lucide-react'
import { useAuthStore } from '@/store/authStore'
import api from '@/services/api'
import { cn } from '@/lib/utils'
import type { Job, JobStatus } from '@/types'

export default function DashboardPage() {
  const user = useAuthStore((s) => s.user)

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-bold text-foreground">Dashboard</h1>
        <p className="mt-1 text-sm text-muted-foreground">Welcome back, {user?.email}</p>
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        <div className="rounded-lg border border-border bg-card p-5 shadow-sm">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">Account</h2>
          <div className="mt-3 space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Plan</span>
              <PlanBadge plan={user?.plan ?? 'free'} />
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Email</span>
              <span className="text-sm font-medium text-foreground">{user?.email}</span>
            </div>
          </div>
          {user?.plan === 'free' && (
            <a
              href="/billing/checkout"
              className={cn(
                'mt-4 flex w-full items-center justify-center gap-1.5 rounded-md',
                'bg-primary px-3 py-2 text-sm font-semibold text-primary-foreground',
                'transition-opacity hover:opacity-90',
              )}
            >
              <Zap className="h-3.5 w-3.5" />
              Upgrade to Pro — $10/mo
              <ArrowUpRight className="h-3.5 w-3.5" />
            </a>
          )}
        </div>

        <div className="lg:col-span-2">
          <SubmitForm />
        </div>
      </div>

      <JobList />
    </div>
  )
}

// ---------------------------------------------------------------------------
// Submit form
// ---------------------------------------------------------------------------

function SubmitForm() {
  const [url, setUrl] = useState('')
  const queryClient = useQueryClient()

  const { mutate, isPending, error, data: submitted } = useMutation<Job, Error, string>({
    mutationFn: async (youtubeUrl) => {
      const res = await api.post<Job>('/v1/transcribe', { youtube_url: youtubeUrl })
      return res.data
    },
    onSuccess: () => {
      setUrl('')
      queryClient.invalidateQueries({ queryKey: ['jobs'] })
    },
  })

  const errorMsg = (error as { response?: { data?: { error?: string } } })
    ?.response?.data?.error ?? (error ? 'Submission failed.' : null)

  return (
    <div className="rounded-lg border border-border bg-card p-5 shadow-sm">
      <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
        New Transcription
      </h2>
      <form
        onSubmit={(e) => { e.preventDefault(); if (url.trim()) mutate(url.trim()) }}
        className="mt-3 flex gap-2"
      >
        <div className="relative flex-1">
          <Youtube className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <input
            type="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://youtube.com/watch?v=..."
            className={cn(
              'w-full rounded-md border border-input bg-background pl-9 pr-3 py-2 text-sm',
              'text-foreground placeholder:text-muted-foreground',
              'focus:outline-none focus:ring-2 focus:ring-ring',
            )}
          />
        </div>
        <button
          type="submit"
          disabled={!url.trim() || isPending}
          className="rounded-md bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground transition-opacity hover:opacity-90 disabled:opacity-50"
        >
          {isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Transcribe'}
        </button>
      </form>

      {errorMsg && (
        <p className="mt-2 text-sm text-destructive">{errorMsg}</p>
      )}
      {submitted && (
        <p className="mt-2 text-sm text-muted-foreground">
          Job <span className="font-mono text-xs">{submitted.id.slice(0, 8)}…</span> queued.
          Results will appear below.
        </p>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Job list
// ---------------------------------------------------------------------------

function JobList() {
  const { data: jobs = [], isFetching } = useQuery<Job[]>({
    queryKey: ['jobs'],
    queryFn: async () => {
      const res = await api.get<Job[]>('/v1/jobs')
      return res.data
    },
    refetchInterval: (query) => {
      const active = query.state.data?.some(
        (j) => j.status === 'pending' || j.status === 'processing',
      )
      return active ? 3000 : false
    },
  })

  if (jobs.length === 0 && !isFetching) return null

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
          Recent Jobs
        </h2>
        {isFetching && <Loader2 className="h-3.5 w-3.5 animate-spin text-muted-foreground" />}
      </div>
      {jobs.map((job) => <JobRow key={job.id} job={job} />)}
    </div>
  )
}

function JobRow({ job: initial }: { job: Job }) {
  const [expanded, setExpanded] = useState(false)

  const { data: job = initial } = useQuery<Job>({
    queryKey: ['job', initial.id],
    queryFn: async () => {
      const res = await api.get<Job>(`/v1/jobs/${initial.id}`)
      return res.data
    },
    initialData: initial,
    refetchInterval: (query) => {
      const s = query.state.data?.status
      return s === 'pending' || s === 'processing' ? 2000 : false
    },
  })

  const hasContent = job.transcript_text || (job.segments && job.segments.length > 0)

  return (
    <div className="rounded-lg border border-border bg-card shadow-sm">
      <button
        onClick={() => hasContent && setExpanded((v) => !v)}
        className={cn(
          'flex w-full items-center gap-3 p-4 text-left',
          hasContent ? 'cursor-pointer' : 'cursor-default',
        )}
      >
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm text-foreground">{job.youtube_url}</p>
          <p className="mt-0.5 text-xs text-muted-foreground">
            {new Date(job.created_at).toLocaleString()}
            {job.language && <> · {job.language.toUpperCase()}</>}
            {job.duration_seconds != null && <> · {Math.round(job.duration_seconds)}s</>}
            {job.whisper_used && <> · Whisper</>}
          </p>
        </div>
        <StatusPill status={job.status} />
        {hasContent && (
          expanded
            ? <ChevronUp className="h-4 w-4 shrink-0 text-muted-foreground" />
            : <ChevronDown className="h-4 w-4 shrink-0 text-muted-foreground" />
        )}
      </button>

      {expanded && hasContent && (
        <div className="border-t border-border px-4 pb-4 pt-3">
          {job.error_message && (
            <p className="mb-3 text-sm text-destructive">{job.error_message}</p>
          )}
          {job.segments && job.segments.length > 0 ? (
            <SegmentTable segments={job.segments} />
          ) : job.transcript_text ? (
            <textarea
              readOnly
              value={job.transcript_text}
              rows={6}
              className="w-full resize-none rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground focus:outline-none"
            />
          ) : null}
        </div>
      )}
    </div>
  )
}

function SegmentTable({ segments }: { segments: NonNullable<Job['segments']> }) {
  return (
    <div className="overflow-hidden rounded-md border border-border">
      <div className="max-h-72 overflow-y-auto">
        <table className="w-full text-sm">
          <thead className="sticky top-0 bg-muted">
            <tr>
              <th className="px-3 py-1.5 text-left text-xs font-semibold text-muted-foreground w-20">Start</th>
              <th className="px-3 py-1.5 text-left text-xs font-semibold text-muted-foreground w-20">End</th>
              <th className="px-3 py-1.5 text-left text-xs font-semibold text-muted-foreground">Text</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {segments.map((seg, i) => (
              <tr key={i} className="hover:bg-muted/40">
                <td className="px-3 py-1.5 font-mono text-xs text-muted-foreground tabular-nums">
                  {formatTs(seg.start)}
                </td>
                <td className="px-3 py-1.5 font-mono text-xs text-muted-foreground tabular-nums">
                  {formatTs(seg.end)}
                </td>
                <td className="px-3 py-1.5 text-foreground">{seg.text}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function formatTs(s: number) {
  const m = Math.floor(s / 60)
  const sec = (s % 60).toFixed(1).padStart(4, '0')
  return `${m}:${sec}`
}

// ---------------------------------------------------------------------------
// Shared components
// ---------------------------------------------------------------------------

const statusConfig: Record<JobStatus, { icon: React.ElementType; label: string; className: string }> = {
  pending:    { icon: Clock,        label: 'Pending',    className: 'bg-secondary text-secondary-foreground' },
  processing: { icon: Loader2,      label: 'Processing', className: 'bg-accent text-accent-foreground' },
  completed:  { icon: CheckCircle2, label: 'Completed',  className: 'bg-primary/10 text-primary' },
  failed:     { icon: XCircle,      label: 'Failed',     className: 'bg-destructive/10 text-destructive' },
}

function StatusPill({ status }: { status: JobStatus }) {
  const { icon: Icon, label, className } = statusConfig[status]
  return (
    <span className={cn('flex shrink-0 items-center gap-1 rounded-full px-2 py-0.5 text-xs font-semibold', className)}>
      <Icon className={cn('h-3 w-3', status === 'processing' && 'animate-spin')} />
      {label}
    </span>
  )
}

function PlanBadge({ plan }: { plan: string }) {
  return (
    <span
      className={cn(
        'rounded-full px-2 py-0.5 text-xs font-semibold capitalize',
        plan === 'free'
          ? 'bg-secondary text-secondary-foreground'
          : 'bg-primary text-primary-foreground',
      )}
    >
      {plan}
    </span>
  )
}
