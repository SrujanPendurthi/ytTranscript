import { useMutation } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { login, signup } from '@/services/authApi'
import { useAuthStore } from '@/store/authStore'
import type { LoginRequest } from '@/types'

function useAuthMutation(apiFn: (data: LoginRequest) => Promise<{ token: string; user: import('@/types').User }>) {
  const setAuth = useAuthStore((s) => s.setAuth)
  const navigate = useNavigate()

  return useMutation({
    mutationFn: apiFn,
    onSuccess: ({ token, user }) => {
      setAuth(token, user)
      navigate('/dashboard', { replace: true })
    },
  })
}

export const useLogin = () => useAuthMutation(login)
export const useSignup = () => useAuthMutation(signup)
