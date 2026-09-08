import { api, clearAccessToken, setAccessToken } from './client'
import type { LoginResponse, User } from '../types/api'

/**
 * Auth API: логин, выход, текущий пользователь (Phase 2 backend).
 */

export async function login(username: string, password: string): Promise<LoginResponse> {
  const data = await api<LoginResponse>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })
  setAccessToken(data.access_token, data.expires_at)
  return data
}

export async function logout(): Promise<void> {
  try {
    await api<void>('/auth/logout', { method: 'POST' })
  } finally {
    clearAccessToken()
  }
}

export async function fetchMe(): Promise<User> {
  return api<User>('/users/me')
}
