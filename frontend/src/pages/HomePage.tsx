import { Logo } from '../components/Logo'

/**
 * Временная стартовая страница проекта QADAM.
 *
 * Подтверждает, что Phase 1 (Project Setup) настроен корректно:
 * React + TypeScript + Tailwind + design-токены применяются.
 * Будет заменена реальным Dashboard в последующих фазах (расписание,
 * уведомления, профиль).
 */
export function HomePage() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-4 bg-background px-4 text-center">
      <Logo className="text-4xl" />
      <p className="max-w-md text-muted">
        Платформа управления учебным процессом. Разработка ведётся поэтапно —
        Phase 1: Project Setup завершён.
      </p>
    </main>
  )
}
