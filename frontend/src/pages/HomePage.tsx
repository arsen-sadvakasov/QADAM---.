import { useAuth } from '../hooks/useAuth'
import { useNotifications } from '../hooks/useNotifications'

/**
 * Домашняя страница (Dashboard, Phase 13.5): приветствие + счётчик
 * непрочитанных. Содержимое будет дополняться: сегодняшние занятия и т.д.
 */
export function HomePage() {
  const { user } = useAuth()
  const { unreadCount } = useNotifications()

  return (
    <section className="flex flex-col gap-4">
      <h1 className="text-lg font-semibold text-text">
        {user ? `Здравствуйте, ${user.full_name}!` : 'Здравствуйте!'}
      </h1>

      {unreadCount > 0 && (
        <a
          href="/notifications"
          className="rounded-lg border border-primary bg-primary/10 p-4 text-sm text-primary"
        >
          У вас {unreadCount} непрочитанных уведомлений — открыть
        </a>
      )}

      <div className="rounded-lg border border-surface bg-surface p-5 text-sm text-muted">
        <p className="font-medium text-text">QADAM — учебная платформа</p>
        <p className="mt-1">
          Расписание, материалы и уведомления колледжа. Приложение можно установить на
          телефон: меню браузера → «Установить».
        </p>
      </div>
    </section>
  )
}
