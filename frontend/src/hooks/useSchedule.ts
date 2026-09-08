import { useQuery } from '@tanstack/react-query'
import { fetchSchedule } from '../api/schedule'
import type { ScheduleFilters } from '../api/schedule'

/**
 * useSchedule — загрузка расписания через react-query (Phase 4/6 backend).
 * Кэш 60 секунд; замены уже применены на backend'е.
 */
export function useSchedule(filters: ScheduleFilters | null) {
  return useQuery({
    queryKey: ['schedule', filters],
    queryFn: () => fetchSchedule(filters as ScheduleFilters),
    enabled: filters !== null && (Boolean(filters.groupId) || Boolean(filters.teacherId)),
    staleTime: 60_000,
    retry: 1,
  })
}
