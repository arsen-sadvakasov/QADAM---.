import { useQuery } from '@tanstack/react-query'
import { fetchGroups, fetchSchedule, fetchSubjects } from '../api/schedule'
import type { Lesson } from '../types/api'

/**
 * useTeacherData — данные дашборда преподавателя (Phase 13.5+):
 * today's lessons по teacher_id (после привязки teacher→user в 16.7),
 * список групп и предметов для отображения "мои дисциплины".
 *
 * ВАЖНО: сейчас backend не связывает teacher-профиль с user-аккаунтом
 * (teachers.user_id есть, но эндпоинта /teachers/me нет). Поэтому
 * дашборд преподавателя показывает: группы, предметы, материалы —
 * а расписание преподавателя доступно через выбор в /schedule.
 */
export function useTeacherData(role: string | undefined) {
  const isTeacher = role === 'teacher'

  const groupsQuery = useQuery({
    queryKey: ['groups'],
    queryFn: fetchGroups,
    select: (data) => data.groups,
    enabled: isTeacher,
    staleTime: 5 * 60_000,
    retry: 1,
  })

  const subjectsQuery = useQuery({
    queryKey: ['subjects'],
    queryFn: fetchSubjects,
    enabled: isTeacher,
    staleTime: 5 * 60_000,
    retry: 1,
  })

  return {
    isTeacher,
    groups: groupsQuery.data ?? [],
    groupsLoading: groupsQuery.isLoading,
    groupsError: groupsQuery.error,
    subjects: subjectsQuery.data ?? [],
    subjectsLoading: subjectsQuery.isLoading,
    subjectsError: subjectsQuery.error,
  }
}

/** Расписание преподавателя на день (для дашборда, когда известен teacher_id). */
export function useTeacherTodayLessons(teacherId: string | null | undefined) {
  return useQuery({
    queryKey: ['schedule', 'teacher-today', teacherId],
    queryFn: () => fetchSchedule({ teacherId: teacherId as string }),
    enabled: Boolean(teacherId),
    staleTime: 60_000,
    retry: 1,
  })
}

/** Разбивает список занятий по датам. */
export function groupLessonsByDate(lessons: Lesson[]): Map<string, Lesson[]> {
  const map = new Map<string, Lesson[]>()
  for (const lesson of lessons) {
    const list = map.get(lesson.date) ?? []
    list.push(lesson)
    map.set(lesson.date, list)
  }
  for (const list of map.values()) {
    list.sort((a, b) => a.start_time.localeCompare(b.start_time))
  }
  return map
}

/** todayKey helper (общий для дашбордов). */
export function todayKey(): string {
  const now = new Date()
  const y = now.getFullYear()
  const m = String(now.getMonth() + 1).padStart(2, '0')
  const d = String(now.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}

/** Форматирование даты для заголовка ("среда, 4 сентября"). */
export function formatDayHeader(d: Date): string {
  return d.toLocaleDateString('ru-RU', { weekday: 'long', day: 'numeric', month: 'long' })
}

/** Сортировка занятий по времени (обычная функция, не хук). */
export function sortedByTime(lessons: Lesson[]): Lesson[] {
  return [...lessons].sort((a, b) => a.start_time.localeCompare(b.start_time))
}
