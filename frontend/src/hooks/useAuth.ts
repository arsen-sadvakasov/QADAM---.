import { useContext } from 'react'
import { AuthContext } from './authContext'
import type { AuthState } from './authContext'

/**
 * Публичный хук доступа к состоянию аутентификации (Phase 13.5).
 * Вынесен в .ts без компонента — react-refresh требование.
 */
export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
