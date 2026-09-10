import type { Lesson } from '../types/api'

/**
 * LessonCard — карточка занятия (единый вид для дашбордов и расписания).
 */
export function LessonCard({ lesson }: { lesson: Lesson }) {
  return (
    <div className="flex items-start gap-3 rounded-lg border border-surface bg-surface p-3">
      <div className="w-12 shrink-0 text-sm font-semibold text-primary">
        {lesson.start_time}
        <div className="text-xs font-normal text-muted">{lesson.end_time}</div>
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-text">{lesson.subject_name}</p>
        <p className="text-xs text-muted">
          {lesson.teacher_name} · каб. {lesson.room_number}
          {lesson.group_name && <span> · {lesson.group_name}</span>}
        </p>
      </div>
      <span className="rounded-full bg-secondary/20 px-2 py-0.5 text-[10px] text-secondary">
        {lessonTypeLabel(lesson.lesson_type)}
      </span>
    </div>
  )
}

// Вынесено в отдельный файл, чтобы файл содержал только компоненты (react-refresh).
function lessonTypeLabel(t: string): string {
  switch (t) {
    case 'lecture':
      return 'Лекция'
    case 'practice':
      return 'Практика'
    case 'lab':
      return 'Лаборатория'
    case 'seminar':
      return 'Семинар'
    default:
      return 'Занятие'
  }
}
