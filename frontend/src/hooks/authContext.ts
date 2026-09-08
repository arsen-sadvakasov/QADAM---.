import { createContext, useMemo, useState } from 'react'
import { fetchMe } from '../api/auth'
import type { User } from '../types/api'

/**
 * AuthContext (Phase 13.5). Провайдер и публичный хук useAuth живут в
 * useAuth.tsx; файл вынесен, чтобы пакет содержал только компоненты
 * (react-refresh).
 */

export interface AuthState {
  user: User | null
  /** true, пока идёт начальная проверка сессии (refresh-cookie). */
  loading: boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => Promise<void>
}

export const AuthContext = createContext<AuthState | null>(null)

/** Внутреннее mutable-состояние провайдера (user + restore). */
export function useSessionState() {
  const [user, setUser] = useState<User | null>(null)

  const restore = useMemo(
    () => async () => {
      try {
        const me = await fetchMe()
        setUser(me)
      } catch {
        setUser(null)
      }
    },
    [],
  )

  return { user, setUser, restore }
}
