import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { clearAccessToken } from '../api/client'
import { login as apiLogin, logout as apiLogout } from '../api/auth'
import { AuthContext, useSessionState } from './authContext'
import type { AuthState } from './authContext'

/**
 * AuthProvider — состояние аутентификации (Phase 2 backend + Phase 13.5).
 * При загрузке проверяет сессию через refresh-cookie; пока идёт проверка —
 * loading=true. Публичный хук useAuth — в useAuth.ts.
 */
export function AuthProvider({ children }: { children: ReactNode }) {
  const session = useSessionState()
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    void session.restore().finally(() => {
      if (!cancelled) setLoading(false)
    })
    return () => {
      cancelled = true
    }
  }, [session])

  const login = useCallback(
    async (username: string, password: string) => {
      const data = await apiLogin(username, password)
      session.setUser(data.user)
    },
    [session],
  )

  const logout = useCallback(async () => {
    await apiLogout()
    clearAccessToken()
    session.setUser(null)
  }, [session])

  const value = useMemo<AuthState>(
    () => ({ user: session.user, loading, login, logout }),
    [session.user, loading, login, logout],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

// Реэкспорт не нужен: useAuth (useAuth.ts) читает AuthContext напрямую.
