/**
 * Типы API QADAM — зеркалируют DTO backend-хендлеров (не изменять имена
 * полей независимо от backend).
 */

export interface User {
  id: string
  username: string
  full_name: string
  role: 'student' | 'teacher' | 'curator' | 'admin'
  language: string
  theme: string
}

export interface LoginResponse {
  access_token: string
  expires_at: string
  user: User
}

export interface Lesson {
  template_id: string
  date: string // YYYY-MM-DD
  group_id: string
  group_name: string
  subject_id: string
  subject_name: string
  teacher_id: string
  teacher_name: string
  room_id: string
  room_number: string
  start_time: string // HH:MM
  end_time: string // HH:MM
  lesson_type: string
}

export interface Notification {
  id: string
  type: string
  title: string
  body: string
  priority: string
  related_entity_type?: string | null
  related_entity_id?: string | null
  created_at: string
  is_read: boolean
  read_at?: string | null
}

export interface NotificationsResponse {
  notifications: Notification[]
  unread_count: number
}

export interface Group {
  id: string
  specialty_id: string
  course_id: string
  curator_id?: string | null
  name: string
}

export interface GroupsResponse {
  groups: Group[]
}
