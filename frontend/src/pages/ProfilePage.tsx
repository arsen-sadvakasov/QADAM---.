import { useAuth } from '../hooks/useAuth'

/**
 * Страница профиля (Phase 13.5): данные пользователя + выход.
 * Настройки языка/темы будут подключены к PATCH /users/me/language и
 * theme_preference в следующей итерации фронтенда.
 */
export function ProfilePage() {
  const { user, logout } = useAuth()

  if (!user) return null

  return (
    <section className="flex flex-col gap-4">
      <h1 className="text-lg font-semibold text-text">Профиль</h1>

      <div className="rounded-lg border border-surface bg-surface p-5">
        <div className="flex items-center gap-4">
          <div className="flex h-14 w-14 items-center justify-center rounded-full bg-secondary/30 text-xl font-bold text-text">
            {user.full_name.charAt(0)}
          </div>
          <div>
            <p className="font-medium text-text">{user.full_name}</p>
            <p className="text-sm text-muted">@{user.username}</p>
            <span className="mt-1 inline-block rounded-full bg-primary/15 px-2 py-0.5 text-[11px] text-primary">
              {roleLabel(user.role)}
            </span>
          </div>
        </div>

        <dl className="mt-4 grid grid-cols-2 gap-2 text-sm">
          <dt className="text-muted">Язык</dt>
          <dd className="text-text">{user.language.toUpperCase()}</dd>
          <dt className="text-muted">Тема</dt>
          <dd className="text-text">{user.theme === 'light' ? 'Светлая' : 'Тёмная'}</dd>
        </dl>
      </div>

      <button
        type="button"
        onClick={() => void logout()}
        className="rounded-lg border border-warning px-4 py-2.5 text-sm font-medium text-warning transition-opacity hover:opacity-80"
      >
        Выйти
      </button>
    </section>
  )
}

function roleLabel(role: string): string {
  switch (role) {
    case 'student':
      return 'Студент'
    case 'teacher':
      return 'Преподаватель'
    case 'curator':
      return 'Куратор'
    case 'admin':
      return 'Администратор'
    default:
      return role
  }
}
