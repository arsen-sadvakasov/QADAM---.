import { api } from './client'
import type { GroupsResponse, Lesson, NotificationsResponse } from '../types/api'

export type { Lesson }

/**
 * API расписания, уведомлений, групп, предметов и материалов
 * (Phase 4/6/8/7 backend).
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

// --- Subjects (Phase 5) ---

export interface Subject {
  id: string
  name: string
  code?: string | null
}

export async function fetchSubjects(): Promise<Subject[]> {
  const data = await api<{ subjects: Subject[] }>('/subjects')
  return data.subjects
}

// --- Materials (Phase 7) ---

export interface Material {
  id: string
  subject_id: string
  subject_name: string
  category: 'lecture' | 'practice' | 'lab' | 'extra'
  title: string
  description?: string | null
  author_name: string
  created_at: string
}

export async function fetchMaterialsBySubject(subjectId: string): Promise<Material[]> {
  const data = await api<{ materials: Material[] }>(`/materials?subject_id=${subjectId}`)
  return data.materials
}

// --- Students (Phase 9; доступ: curator своей группы, admin) ---

export interface Student {
  id: string
  user_id?: string | null
  group_id: string
  status: string
  full_name?: string | null
  email?: string | null
}

export async function fetchStudentsByGroup(groupId: string): Promise<Student[]> {
  const data = await api<{ students: Student[] }>(`/students?group_id=${groupId}`)
  return data.students
}
