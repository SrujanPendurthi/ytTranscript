import { Outlet, NavLink, useNavigate } from 'react-router-dom'
import { LogOut, Zap } from 'lucide-react'
import { useAuthStore } from '@/features/auth'
import { cn } from '@/utils'

export default function AppLayout() {
  const { user, logout } = useAuthStore()
  const navigate = useNavigate()

  const handleLogout = () => {
    logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="min-h-screen bg-background">
      <header className="bg-white border-b border-border shadow-sm">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
          <NavLink to="/dashboard" className="flex items-center gap-2.5">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-indigo-600 to-blue-500 shadow-sm">
              <Zap className="h-4 w-4 text-white" />
            </div>
            <span className="text-base font-bold text-foreground">TranscriptFlow</span>
          </NavLink>

          <nav className="hidden gap-1 sm:flex">
            {[{ to: '/dashboard', label: 'Dashboard' }, { to: '/developer', label: 'Developer' }].map(({ to, label }) => (
              <NavLink
                key={to}
                to={to}
                className={({ isActive }) =>
                  cn(
                    'rounded-lg px-3 py-1.5 text-sm font-medium transition-colors',
                    isActive
                      ? 'bg-accent text-primary font-semibold'
                      : 'text-muted-foreground hover:bg-secondary hover:text-foreground',
                  )
                }
              >
                {label}
              </NavLink>
            ))}
          </nav>

          <div className="flex items-center gap-3">
            <div className="hidden flex-col items-end sm:flex">
              <span className="text-sm font-medium text-foreground">{user?.email}</span>
              <PlanBadge plan={user?.plan ?? 'free'} />
            </div>
            <button
              onClick={handleLogout}
              aria-label="Logout"
              className="rounded-lg p-2 text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
            >
              <LogOut className="h-4 w-4" />
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-6 py-8">
        <Outlet />
      </main>
    </div>
  )
}

function PlanBadge({ plan }: { plan: string }) {
  return (
    <span
      className={cn(
        'rounded-full px-2 py-0.5 text-xs font-semibold capitalize',
        plan === 'free'
          ? 'bg-secondary text-secondary-foreground'
          : 'bg-gradient-to-r from-indigo-600 to-blue-500 text-white',
      )}
    >
      {plan}
    </span>
  )
}
