import { Zap } from 'lucide-react'

/** Shared input styling for the auth forms — matches the dashboard composer. */
export const authInputClass =
  'w-full rounded-xl border border-input bg-card px-3.5 py-2.5 text-sm text-foreground ' +
  'placeholder:text-muted-foreground/70 outline-none transition ' +
  'focus:border-primary focus:ring-4 focus:ring-primary/15'

export function AuthShell({
  title,
  subtitle,
  children,
  footer,
  note,
}: {
  title: string
  subtitle: string
  children: React.ReactNode
  footer: React.ReactNode
  note?: React.ReactNode
}) {
  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-background px-4">
      {/* ambient brand glow */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-x-0 top-0 h-96 bg-[radial-gradient(700px_300px_at_50%_-80px,rgba(99,102,241,0.13),transparent_70%)]"
      />

      <div className="relative w-full max-w-sm">
        <div className="mb-6 flex flex-col items-center gap-3 text-center">
          <div className="flex items-center gap-2.5">
            <span className="grid h-9 w-9 place-items-center rounded-xl bg-gradient-to-br from-indigo-600 to-violet-500 shadow-[0_6px_16px_-6px_rgba(91,87,240,0.7)]">
              <Zap className="h-[18px] w-[18px] text-white" />
            </span>
            <span className="text-xl font-bold tracking-tight text-foreground">TranscriptFlow</span>
          </div>
          <div>
            <h1 className="text-lg font-semibold tracking-tight text-foreground">{title}</h1>
            <p className="mt-0.5 text-sm text-muted-foreground">{subtitle}</p>
          </div>
        </div>

        <div className="rounded-2xl border border-border bg-card p-6 shadow-[0_2px_4px_rgba(20,22,45,0.04),0_26px_50px_-24px_rgba(48,44,120,0.28)]">
          {children}
        </div>

        {note && <div className="mt-4 text-center">{note}</div>}

        <p className="mt-5 text-center text-sm text-muted-foreground">{footer}</p>
      </div>
    </div>
  )
}
