import { useQuery } from '@tanstack/react-query'
import { fetchStudentsByGroup } from '../api/schedule'
import type { Student } from '../types/api'

/**
 * useCuratorStudents — студенты группы куратора (Phase 9 backend:
 * GET /students?group_id=, доступ curator своей группы / admin).
 * groupId передаёт вызывающий (например, из профиля куратора → его группа).
 */
export function useCuratorStudents(groupId: string | null | undefined) {
  return useQuery({
    queryKey: ['students', groupId],
    queryFn: () => fetchStudentsByGroup(groupId as string),
    enabled: Boolean(groupId),
    staleTime: 2 * 60_000,
    retry: 1,
  })
}

export type { Student }
