/**
 * Базовый API-клиент: fetch с JWT access-токеном и авто-refresh при 401
 * (refresh-токен живёт в httpOnly cookie, ставится backend'ом).
 */

const API_BASE = '/api/v1'

export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

// --- Токен в памяти (не localStorage — XSS-безопасность, раздел 28) ---
let accessToken: string | null = null
let tokenExpiresAt: number = 0

export function setAccessToken(token: string, expiresAtIso: string): void {
  accessToken = token
  tokenExpiresAt = new Date(expiresAtIso).getTime()
}

export function clearAccessToken(): void {
  accessToken = null
  tokenExpiresAt = 0
}

function hasValidToken(): boolean {
  return accessToken !== null && Date.now() < tokenExpiresAt - 5000
}

// --- Refresh: одна параллельная попытка на все запросы ---
let refreshPromise: Promise<boolean> | null = null

async function refreshAccessToken(): Promise<boolean> {
  if (refreshPromise) return refreshPromise

  refreshPromise = (async () => {
    try {
      const resp = await fetch(`${API_BASE}/auth/refresh`, {
        method: 'POST',
        credentials: 'include',
      })
      if (!resp.ok) return false
      const data: LoginResponseType = await resp.json()
      setAccessToken(data.access_token, data.expires_at)
      return true
    } catch {
      return false
    } finally {
      refreshPromise = null
    }
  })()
  return refreshPromise
}

// Локальный тип, чтобы не зациклить импорты.
interface LoginResponseType {
  access_token: string
  expires_at: string
  user: unknown
}

/**
 * Выполняет запрос к API. При 401 пробует обновить токен и повторить один раз.
 */
export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  async function doFetch(): Promise<Response> {
    const headers = new Headers(init.headers)
    if (accessToken) headers.set('Authorization', `Bearer ${accessToken}`)
    if (init.body && !headers.has('Content-Type')) {
      headers.set('Content-Type', 'application/json')
    }
    return fetch(`${API_BASE}${path}`, { ...init, headers, credentials: 'include' })
  }

  let resp = await doFetch()

  if (resp.status === 401 && path !== '/auth/login') {
    const refreshed = await refreshAccessToken()
    if (refreshed) {
      resp = await doFetch()
    }
  }

  if (!resp.ok) {
    let message = `HTTP ${resp.status}`
    try {
      const body = (await resp.json()) as { error?: string }
      if (body.error) message = body.error
    } catch {
      // тело не JSON — оставляем HTTP-код
    }
    throw new ApiError(resp.status, message)
  }

  if (resp.status === 204) return undefined as T
  return (await resp.json()) as T
}

export { API_BASE, hasValidToken }
