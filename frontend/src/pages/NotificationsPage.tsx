import { useNotifications } from '../hooks/useNotifications'

/**
 * Страница уведомлений (Phase 8 backend + Phase 13.5 фронтенд).
 */
const priorityStyles: Record<string, string> = {
  low: 'bg-muted/20 text-muted',
  normal: 'bg-secondary/20 text-secondary',
  high: 'bg-primary/15 text-primary',
  critical: 'bg-warning/15 text-warning',
}

export function NotificationsPage() {
  const { notifications, isLoading, error, markRead, markReadPending } = useNotifications()

  return (
    <section className="flex flex-col gap-3">
      <h1 className="text-lg font-semibold text-text">Уведомления</h1>

      {isLoading && <p className="text-sm text-muted">Загрузка…</p>}
      {error && (
        <p role="alert" className="rounded-lg bg-warning/10 px-3 py-2 text-sm text-warning">
          Не удалось загрузить уведомления. Попробуйте позже.
        </p>
      )}

      {!isLoading && !error && notifications.length === 0 && (
        <div className="rounded-lg border border-surface bg-surface p-6 text-center text-sm text-muted">
          Уведомлений нет
        </div>
      )}

      <ul className="flex flex-col gap-2">
        {notifications.map((n) => (
          <li
            key={n.id}
            className={`rounded-lg border border-surface bg-surface p-4 ${
              n.is_read ? 'opacity-60' : ''
            }`}
          >
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <span
                    className={`rounded-full px-2 py-0.5 text-[10px] ${
                      priorityStyles[n.priority] ?? priorityStyles.normal
                    }`}
                  >
                    {n.type}
                  </span>
                  {!n.is_read && (
                    <span className="h-2 w-2 rounded-full bg-primary" aria-label="Непрочитано" />
                  )}
                </div>
                <p className="mt-1 text-sm font-medium text-text">{n.title}</p>
                <p className="mt-0.5 text-sm text-muted">{n.body}</p>
                <p className="mt-1 text-[11px] text-muted">{formatDate(n.created_at)}</p>
              </div>
              {!n.is_read && (
                <button
                  type="button"
                  disabled={markReadPending}
                  onClick={() => markRead(n.id)}
                  className="shrink-0 rounded-lg border border-primary px-2.5 py-1 text-xs text-primary transition-opacity disabled:opacity-50"
                >
                  Прочитано
                </button>
              )}
            </div>
          </li>
        ))}
      </ul>
    </section>
  )
}

function formatDate(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleString('ru-RU', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })
}
