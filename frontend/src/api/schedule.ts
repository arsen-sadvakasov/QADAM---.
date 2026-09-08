import { api } from './client'
import type { GroupsResponse, Lesson, NotificationsResponse } from '../types/api'

/**
 * API расписания, уведомлений и групп (Phase 4/6/8 backend).
 */

export interface ScheduleFilters {
  groupId?: string
  teacherId?: string
  from?: string // YYYY-MM-DD
  to?: string // YYYY-MM-DD
}

export async function fetchSchedule(filters: ScheduleFilters): Promise<Lesson[]> {
  const params = new URLSearchParams()
  if (filters.groupId) params.set('group_id', filters.groupId)
  if (filters.teacherId) params.set('teacher_id', filters.teacherId)
  if (filters.from) params.set('from', filters.from)
  if (filters.to) params.set('to', filters.to)

  const data = await api<{ lessons: Lesson[] }>(`/schedules?${params.toString()}`)
  return data.lessons
}

export async function fetchNotifications(isRead?: boolean): Promise<NotificationsResponse> {
  const suffix = isRead === undefined ? '' : `&is_read=${isRead}`
  return api<NotificationsResponse>(`/notifications?limit=50${suffix}`)
}

export async function markNotificationRead(id: string): Promise<void> {
  await api<void>(`/notifications/${id}/read`, { method: 'PATCH' })
}

export async function fetchGroups(): Promise<GroupsResponse> {
  return api<GroupsResponse>('/groups')
}
