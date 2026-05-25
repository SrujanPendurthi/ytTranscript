import { Navigate, Outlet } from 'react-router-dom'
import { useAuthStore } from '@/store/authStore'

// Renders child routes when authenticated; redirects to /login otherwise.
// Used as a layout route so nested <Route> elements share the auth gate.
export default function ProtectedRoute() {
  const token = useAuthStore((s) => s.token)
  if (!token) return <Navigate to="/login" replace />
  return <Outlet />
}
