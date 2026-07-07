import { Outlet, NavLink, useNavigate } from 'react-router-dom'
import { LogOut, Zap, KeyRound } from 'lucide-react'
import { useAuthStore } from '@/features/auth'
import { cn } from '@/utils'

const NAV = [
  { to: '/dashboard', label: 'Dashboard' },
  { to: '/developer', label: 'Developer' },
]

export default function AppLayout() {
  const { user, logout } = useAuthStore()
  const navigate = useNavigate()

  const handleLogout = () => {
    logout()
    navigate('/login', { replace: true })
  }

  const initials = (user?.email ?? '?').slice(0, 2).toUpperCase()

  return (
    <div className="min-h-screen bg-background">
      <header className="sticky top-0 z-20 border-b border-border bg-background/75 backdrop-blur-md">
        <div className="mx-auto flex max-w-6xl items-center gap-5 px-6 py-3">
          <NavLink to="/dashboard" className="flex items-center gap-2.5">
            <span className="grid h-[29px] w-[29px] place-items-center rounded-[9px] bg-gradient-to-br from-indigo-600 to-violet-500 shadow-[0_6px_16px_-6px_rgba(91,87,240,0.7)]">
              <Zap className="h-[15px] w-[15px] text-white" />
            </span>
            <span className="text-[15px] font-bold tracking-tight text-foreground">TranscriptFlow</span>
          </NavLink>

          <nav className="ml-2 flex gap-0.5">
            {NAV.map(({ to, label }) => (
              <NavLink
                key={to}
                to={to}
                className={({ isActive }) =>
                  cn(
                    'rounded-[9px] px-3 py-1.5 text-sm transition-colors',
                    isActive
                      ? 'bg-accent text-accent-foreground'
                      : 'text-muted-foreground hover:bg-secondary hover:text-foreground',
                  )
                }
              >
                {label}
              </NavLink>
            ))}
          </nav>

          <div className="ml-auto flex items-center gap-3">
            <NavLink
              to="/developer"
              title="API keys"
              aria-label="API keys"
              className="grid h-[34px] w-[34px] place-items-center rounded-[9px] border border-border bg-card text-muted-foreground transition-colors hover:border-input hover:text-primary"
            >
              <KeyRound className="h-4 w-4" />
            </NavLink>

            <div className="hidden flex-col items-end leading-tight sm:flex">
              <span className="text-[13px] text-foreground">{user?.email}</span>
              <PlanBadge plan={user?.plan ?? 'free'} />
            </div>

            <span className="grid h-[34px] w-[34px] place-items-center rounded-full bg-gradient-to-br from-indigo-100 to-violet-100 text-[12.5px] font-bold text-indigo-700 ring-1 ring-border">
              {initials}
            </span>

            <button
              onClick={handleLogout}
              aria-label="Log out"
              className="grid h-[34px] w-[34px] place-items-center rounded-[9px] border border-border bg-card text-muted-foreground transition-colors hover:border-input hover:text-foreground"
            >
              <LogOut className="h-4 w-4" />
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-6 pb-20 pt-6">
        <Outlet />
      </main>
    </div>
  )
}

function PlanBadge({ plan }: { plan: string }) {
  return (
    <span
      className={cn(
        'rounded-full px-2 py-0.5 font-mono text-[10px] uppercase tracking-wider',
        plan === 'free'
          ? 'border border-border text-muted-foreground'
          : 'bg-gradient-to-r from-indigo-600 to-violet-500 text-white',
      )}
    >
      {plan}
    </span>
  )
}
