// Public API for the auth feature — import from '@/features/auth', not deep paths.
export { default as LoginPage } from './components/LoginPage'
export { default as SignupPage } from './components/SignupPage'
export { default as ProtectedRoute } from './components/ProtectedRoute'
export { useAuthStore } from './hooks/authStore'
export { useLogin, useSignup } from './hooks/useAuthMutations'
export type { Plan, User, AuthResponse, LoginRequest } from './types'
