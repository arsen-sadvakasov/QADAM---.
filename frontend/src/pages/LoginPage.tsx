import { useState } from 'react'
import type { FormEvent } from 'react'
import { useAuth } from '../hooks/useAuth'
import { ApiError } from '../api/client'

/**
 * Страница входа (Phase 2 backend + Phase 13.5 фронтенд).
 */
export function LoginPage() {
  const { login } = useAuth()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setPending(true)
    try {
      await login(username, password)
      // перезагрузка не нужна: AuthProvider обновит состояние
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setError('Неверный логин или пароль')
      } else if (err instanceof ApiError && err.status === 429) {
        setError('Слишком много попыток. Подождите минуту.')
      } else {
        setError('Не удалось войти. Попробуйте позже.')
      }
    } finally {
      setPending(false)
    }
  }

  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-6 bg-background px-4">
      <h1 className="text-3xl font-bold tracking-tight text-primary">QADAM</h1>
      <form
        onSubmit={handleSubmit}
        className="w-full max-w-sm rounded-xl border border-surface bg-surface p-6 flex flex-col gap-4"
      >
        <div>
          <label htmlFor="username" className="mb-1 block text-sm text-muted">
            Логин
          </label>
          <input
            id="username"
            type="text"
            autoComplete="username"
            required
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            className="w-full rounded-lg border border-muted/30 bg-background px-3 py-2 text-text outline-none focus:border-primary"
          />
        </div>
        <div>
          <label htmlFor="password" className="mb-1 block text-sm text-muted">
            Пароль
          </label>
          <input
            id="password"
            type="password"
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="w-full rounded-lg border border-muted/30 bg-background px-3 py-2 text-text outline-none focus:border-primary"
          />
        </div>

        {error && (
          <p role="alert" className="rounded-lg bg-warning/10 px-3 py-2 text-sm text-warning">
            {error}
          </p>
        )}

        <button
          type="submit"
          disabled={pending}
          className="rounded-lg bg-primary px-4 py-2.5 font-semibold text-background transition-opacity disabled:opacity-50"
        >
          {pending ? 'Вход…' : 'Войти'}
        </button>
      </form>
    </main>
  )
}
