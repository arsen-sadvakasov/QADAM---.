import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { fetchSchedule } from '../api/schedule'
import type { Lesson, ScheduleFilters } from '../api/schedule'

export interface DayScheduleResult {
  lessons: Lesson[]
  /** ISO-дата дня (YYYY-MM-DD) */
  dateKey: string
}

function dateKey(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

/**
 * useTodaySchedule — занятия на сегодня (с заменами, применёнными backend'ом).
 * Если сегодня воскресенье — показывает понедельник следующей недели.
 * enabled=false, пока фильтры не определены (роль пользователя ещё не ясна).
 */
export function useTodaySchedule(filters: ScheduleFilters | null | undefined) {
  const enabled = Boolean(
    filters && (filters.groupId || filters.teacherId),
  )

  const { todayKey, displayDate } = useMemo(() => {
    const now = new Date()
    // Воскресенье (0) → показываем завтрашний понедельник.
    if (now.getDay() === 0) {
      const tomorrow = new Date(now)
      tomorrow.setDate(now.getDate() + 1)
      return { todayKey: dateKey(now), displayDate: tomorrow }
    }
    return { todayKey: dateKey(now), displayDate: now }
  }, [/* день вычисляется при монтировании */])

  const query = useQuery({
    queryKey: ['schedule', 'today', filters],
    queryFn: () => fetchSchedule(filters as ScheduleFilters),
    enabled,
    staleTime: 60_000,
    retry: 1,
  })

  const lessons = useMemo(() => {
    if (!query.data) return []
    return query.data
      .filter((l) => l.date === dateKey(displayDate))
      .sort((a, b) => a.start_time.localeCompare(b.start_time))
  }, [query.data, displayDate])

  return {
    lessons,
    dateKey: todayKey,
    displayDate,
    isSunday: new Date().getDay() === 0,
    isLoading: enabled && query.isLoading,
    error: query.error,
    enabled,
  }
}
