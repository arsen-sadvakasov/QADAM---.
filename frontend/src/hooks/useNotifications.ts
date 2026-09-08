import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { fetchNotifications, markNotificationRead } from '../api/schedule'

/**
 * useNotifications — уведомления пользователя (Phase 8 backend).
 * unreadCount используется бейджем в навигации.
 */
export function useNotifications() {
  const query = useQuery({
    queryKey: ['notifications'],
    queryFn: () => fetchNotifications(),
    staleTime: 30_000,
    retry: 1,
  })

  const queryClient = useQueryClient()
  const markRead = useMutation({
    mutationFn: (id: string) => markNotificationRead(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['notifications'] })
    },
  })

  return {
    notifications: query.data?.notifications ?? [],
    unreadCount: query.data?.unread_count ?? 0,
    isLoading: query.isLoading,
    error: query.error,
    markRead: markRead.mutate,
    markReadPending: markRead.isPending,
  }
}
