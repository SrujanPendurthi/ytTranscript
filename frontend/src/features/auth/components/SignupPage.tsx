import { useState } from 'react'
import { Link } from 'react-router-dom'
import { ArrowRight, Loader2, Check } from 'lucide-react'
import { useSignup } from '../hooks/useAuthMutations'
import { AuthShell, authInputClass } from './AuthShell'

export default function SignupPage() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [clientError, setClientError] = useState<string | null>(null)
  const { mutate, isPending, error } = useSignup()

  const serverErrorMsg =
    (error as { response?: { data?: { error?: string } } })?.response?.data?.error ??
    (error ? 'Sign up failed. Please try again.' : null)

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setClientError(null)
    if (password.length < 8) return setClientError('Password must be at least 8 characters.')
    if (password !== confirm) return setClientError('Passwords do not match.')
    mutate({ email: email.trim(), password })
  }

  const displayError = clientError ?? serverErrorMsg

  return (
    <AuthShell
      title="Create your account"
      subtitle="Start transcribing in under a minute"
      note={
        <div className="flex flex-wrap items-center justify-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
          {['Free forever tier', 'No credit card', '100 / month'].map((t) => (
            <span key={t} className="inline-flex items-center gap-1">
              <Check className="h-3 w-3 text-emerald-500" /> {t}
            </span>
          ))}
        </div>
      }
      footer={
        <>
          Have an account?{' '}
          <Link to="/login" className="font-medium text-primary underline-offset-4 hover:underline">
            Sign in
          </Link>
        </>
      }
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        <Field label="Email">
          <input
            type="email"
            autoComplete="email"
            required
            autoFocus
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@example.com"
            className={authInputClass}
          />
        </Field>

        <Field label="Password">
          <input
            type="password"
            autoComplete="new-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Min. 8 characters"
            className={authInputClass}
          />
        </Field>

        <Field label="Confirm password">
          <input
            type="password"
            autoComplete="new-password"
            required
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            placeholder="••••••••"
            className={authInputClass}
          />
        </Field>

        {displayError && (
          <p className="rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive">{displayError}</p>
        )}

        <button
          type="submit"
          disabled={isPending}
          className="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-gradient-to-br from-indigo-600 to-violet-500 px-4 py-2.5 text-sm font-semibold text-white shadow-[0_10px_22px_-8px_rgba(91,87,240,0.6)] transition hover:brightness-105 active:translate-y-px disabled:opacity-60"
        >
          {isPending ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" /> Creating account…
            </>
          ) : (
            <>
              Create account <ArrowRight className="h-4 w-4" />
            </>
          )}
        </button>
      </form>
    </AuthShell>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1.5">
      <label className="block text-sm font-medium text-foreground">{label}</label>
      {children}
    </div>
  )
}
