import type { ReactNode } from 'react'
import { useAuth } from '../hooks/useAuth'
import { LoginPage } from '../pages/LoginPage'

/**
 * Защищённый маршрут: пока сессия проверяется — спиннер;
 * не авторизован — страница входа; авторизован — контент.
 */
export function ProtectedRoute({ children }: { children: ReactNode }) {
  const { user, loading } = useAuth()

  if (loading) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-background text-muted">
        Загрузка…
      </main>
    )
  }

  if (!user) return <LoginPage />

  return <>{children}</>
}
