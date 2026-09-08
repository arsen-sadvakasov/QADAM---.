import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { fetchGroups } from '../api/schedule'
import { useSchedule } from '../hooks/useSchedule'
import type { Lesson } from '../types/api'

/**
 * Страница расписания (Phase 4/6 backend + Phase 13.5 фронтенд).
 * Выбор группы → расписание на неделю (пн–вс), замены подсвечены — они уже
 * применены на backend'е (отменённые скрыты, заменённые поля подставлены).
 */
export function SchedulePage() {
  const groupsQuery = useQuery({
    queryKey: ['groups'],
    queryFn: fetchGroups,
    staleTime: 5 * 60_000,
    retry: 1,
  })

  // TODO(роли): для преподавателя — выбор по teacher_id; в MVP студент/куратор
  // выбирают группу вручную (персональная привязка придёт с Phase 9 UI).
  const [selectedGroupId, setSelectedGroupId] = useState<string | null>(null)
  // Derived state: если пользователь ещё не выбрал группу — берём первую
  // из загруженных (без useEffect — предупреждение react-refresh).
  const groupId = selectedGroupId ?? groupsQuery.data?.groups[0]?.id ?? null
  const setGroupId = setSelectedGroupId
  const filters = groupId ? { groupId } : null
  const schedule = useSchedule(filters)

  const weekStart = getMonday(new Date())
  const days = Array.from({ length: 7 }, (_, i) => {
    const d = new Date(weekStart)
    d.setDate(d.getDate() + i)
    return d
  })

  const lessonsByDate = groupByDate(schedule.data ?? [])

  return (
    <section className="flex flex-col gap-4">
      <h1 className="text-lg font-semibold text-text">Расписание</h1>

      <label className="flex flex-col gap-1 text-sm text-muted">
        Группа
        <select
          value={groupId ?? ''}
          onChange={(e) => setGroupId(e.target.value)}
          className="rounded-lg border border-muted/30 bg-surface px-3 py-2 text-text outline-none focus:border-primary"
        >
          <option value="" disabled>
            {groupsQuery.isLoading ? 'Загрузка…' : 'Выберите группу'}
          </option>
          {groupsQuery.data?.groups.map((g) => (
            <option key={g.id} value={g.id}>
              {g.name}
            </option>
          ))}
        </select>
      </label>

      {schedule.isLoading && <p className="text-sm text-muted">Загрузка расписания…</p>}
      {schedule.error && (
        <p role="alert" className="rounded-lg bg-warning/10 px-3 py-2 text-sm text-warning">
          Не удалось загрузить расписание. Попробуйте позже.
        </p>
      )}

      {groupId && !schedule.isLoading && (
        <div className="flex flex-col gap-3">
          {days.map((day) => {
            const key = toDateKey(day)
            const lessons = lessonsByDate.get(key) ?? []
            return (
              <article key={key} className="rounded-lg border border-surface bg-surface">
                <header className="border-b border-surface px-4 py-2 text-sm font-medium text-muted">
                  {formatDay(day)}
                </header>
                {lessons.length === 0 ? (
                  <p className="px-4 py-3 text-sm text-muted">Нет занятий</p>
                ) : (
                  <ul className="divide-y divide-surface">
                    {lessons.map((lesson) => (
                      <LessonRow key={lesson.template_id + lesson.date} lesson={lesson} />
                    ))}
                  </ul>
                )}
              </article>
            )
          })}
        </div>
      )}
    </section>
  )
}

function LessonRow({ lesson }: { lesson: Lesson }) {
  return (
    <li className="flex items-start gap-3 px-4 py-3">
      <div className="w-12 shrink-0 text-sm font-semibold text-primary">
        {lesson.start_time}
        <div className="text-xs font-normal text-muted">{lesson.end_time}</div>
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-text">{lesson.subject_name}</p>
        <p className="text-xs text-muted">
          {lesson.teacher_name} · каб. {lesson.room_number}
        </p>
      </div>
      <span className="rounded-full bg-secondary/20 px-2 py-0.5 text-[10px] text-secondary">
        {lesson.lesson_type}
      </span>
    </li>
  )
}

// --- helpers ---

function getMonday(date: Date): Date {
  const d = new Date(date)
  const day = d.getDay()
  const diff = day === 0 ? -6 : 1 - day
  d.setDate(d.getDate() + diff)
  d.setHours(0, 0, 0, 0)
  return d
}

function toDateKey(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

const dayNames = ['Воскресенье', 'Понедельник', 'Вторник', 'Среда', 'Четверг', 'Пятница', 'Суббота']

function formatDay(d: Date): string {
  const today = new Date()
  const isToday = toDateKey(d) === toDateKey(today)
  const label = `${dayNames[d.getDay()]}, ${d.getDate()}.${String(d.getMonth() + 1).padStart(2, '0')}`
  return isToday ? `Сегодня — ${label}` : label
}

function groupByDate(lessons: Lesson[]): Map<string, Lesson[]> {
  const map = new Map<string, Lesson[]>()
  for (const lesson of lessons) {
    const list = map.get(lesson.date) ?? []
    list.push(lesson)
    map.set(lesson.date, list)
  }
  for (const list of map.values()) {
    list.sort((a, b) => a.start_time.localeCompare(b.start_time))
  }
  return map
}
