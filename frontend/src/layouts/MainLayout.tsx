import { NavLink, Outlet } from 'react-router-dom'
import { Logo } from '../components/Logo'
import { useNotifications } from '../hooks/useNotifications'

/**
 * Основной layout (Phase 13 — Mobile First, раздел 8 спецификации):
 * контент сверху, нижняя таб-навигация снизу — основной паттерн для
 * мобильных устройств (большинство пользователей заходят с телефона).
 * На десктопе (sm+) навигация переезжает в верхнюю панель.
 */

const tabs = [
  { to: '/', label: 'Главная', icon: '🏠', end: true },
  { to: '/schedule', label: 'Расписание', icon: '📅', end: false },
  { to: '/notifications', label: 'Уведомления', icon: '🔔', end: false },
  { to: '/profile', label: 'Профиль', icon: '👤', end: false },
]

export function MainLayout() {
  const { unreadCount } = useNotifications()

  return (
    <div className="flex min-h-screen flex-col bg-background">
      {/* Верхняя панель: логотип (на мобильных компактная) */}
      <header className="sticky top-0 z-10 border-b border-surface bg-background/95 backdrop-blur">
        <div className="mx-auto flex h-14 max-w-3xl items-center justify-between px-4">
          <Logo className="text-xl" />
          {/* Десктопная навигация (sm+) */}
          <nav className="hidden gap-6 sm:flex" aria-label="Основная навигация">
            {tabs.map((tab) => (
              <NavLink
                key={tab.to}
                to={tab.to}
                end={tab.end}
                className={({ isActive }) =>
                  `relative text-sm transition-colors ${
                    isActive ? 'text-primary' : 'text-muted hover:text-text'
                  }`
                }
              >
                {tab.label}
                {tab.to === '/notifications' && unreadCount > 0 && (
                  <span className="ml-1 rounded-full bg-warning px-1.5 py-0.5 text-[10px] text-background">
                    {unreadCount}
                  </span>
                )}
              </NavLink>
            ))}
          </nav>
        </div>
      </header>

      {/* Контент с отступом под нижнюю таб-навигацию на мобильных */}
      <main className="mx-auto w-full max-w-3xl flex-1 px-4 pb-20 pt-4 sm:pb-8">
        <Outlet />
      </main>

      {/* Нижняя таб-навигация (мобильные, < sm) — safe-area для iPhone */}
      <nav
        className="fixed inset-x-0 bottom-0 z-10 border-t border-surface bg-background/95 backdrop-blur sm:hidden"
        style={{ paddingBottom: 'env(safe-area-inset-bottom)' }}
        aria-label="Нижняя навигация"
      >
        <div className="grid grid-cols-4">
          {tabs.map((tab) => (
            <NavLink
              key={tab.to}
              to={tab.to}
              end={tab.end}
              className={({ isActive }) =>
                `relative flex flex-col items-center gap-0.5 py-2 text-[11px] transition-colors ${
                  isActive ? 'text-primary' : 'text-muted'
                }`
              }
            >
              <span aria-hidden="true" className="text-lg leading-none">
                {tab.icon}
              </span>
              {tab.label}
              {tab.to === '/notifications' && unreadCount > 0 && (
                <span className="absolute right-4 top-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-warning px-1 text-[9px] font-bold text-background">
                  {unreadCount > 9 ? '9+' : unreadCount}
                </span>
              )}
            </NavLink>
          ))}
        </div>
      </nav>
    </div>
  )
}
