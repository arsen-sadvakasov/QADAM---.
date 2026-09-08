import { Link } from 'react-router-dom'

/** Страница 404 — маршрут не найден. */
export function NotFoundPage() {
  return (
    <section className="flex flex-col items-center gap-4 py-16 text-center">
      <p className="text-5xl font-bold text-primary">404</p>
      <p className="text-sm text-muted">Страница не найдена</p>
      <Link to="/" className="text-sm text-primary underline">
        На главную
      </Link>
    </section>
  )
}
