/**
 * Единый компонент логотипа QADAM.
 *
 * Абстрагирует текущее отсутствие финального SVG-логотипа (см. раздел 7
 * спецификации). Пока рендерит текстовый placeholder на акцентном цвете —
 * при готовности финального лого меняется только тело этого компонента,
 * без изменений в местах использования.
 */
export function Logo({ className = '' }: { className?: string }) {
  return (
    <span
      className={`font-bold tracking-tight text-primary ${className}`}
      aria-label="QADAM"
    >
      QADAM
    </span>
  )
}
