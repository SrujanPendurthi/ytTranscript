import { Outlet, NavLink, useNavigate } from 'react-router-dom'
import { LogOut, Zap } from 'lucide-react'
import { useAuthStore } from '@/store/authStore'
import { cn } from '@/lib/utils'

export default function AppLayout() {
  const { user, logout } = useAuthStore()
  const navigate = useNavigate()

  const handleLogout = () => {
    logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Top navigation bar */}
      <header className="border-b border-border bg-card">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-3">
          {/* Logo */}
          <NavLink to="/dashboard" className="flex items-center gap-2 font-bold text-foreground">
            <Zap className="h-5 w-5 text-primary" />
            TranscriptFlow
          </NavLink>

          {/* Nav links */}
          <nav className="hidden gap-6 sm:flex">
            {[{ to: '/dashboard', label: 'Dashboard' }, { to: '/developer', label: 'Developer' }].map(({ to, label }) => (
              <NavLink
                key={to}
                to={to}
                className={({ isActive }) =>
                  cn(
                    'text-sm font-medium transition-colors',
                    isActive
                      ? 'text-foreground'
                      : 'text-muted-foreground hover:text-foreground',
                  )
                }
              >
                {label}
              </NavLink>
            ))}
          </nav>

          {/* User controls */}
          <div className="flex items-center gap-4">
            <div className="hidden flex-col items-end sm:flex">
              <span className="text-sm font-medium text-foreground">{user?.email}</span>
              <PlanBadge plan={user?.plan ?? 'free'} />
            </div>
            <button
              onClick={handleLogout}
              aria-label="Logout"
              className="rounded-md p-2 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
            >
              <LogOut className="h-4 w-4" />
            </button>
          </div>
        </div>
      </header>

      {/* Page content */}
      <main className="mx-auto max-w-6xl px-6 py-8">
        <Outlet />
      </main>
    </div>
  )
}

function PlanBadge({ plan }: { plan: string }) {
  const styles: Record<string, string> = {
    free: 'bg-secondary text-secondary-foreground',
    pro: 'bg-primary text-primary-foreground',
    enterprise: 'bg-primary text-primary-foreground',
  }
  return (
    <span
      className={cn(
        'rounded-full px-2 py-0.5 text-xs font-semibold capitalize',
        styles[plan] ?? styles.free,
      )}
    >
      {plan}
    </span>
  )
}
