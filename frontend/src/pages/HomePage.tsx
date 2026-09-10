import { useAuth } from '../hooks/useAuth'
import { useNotifications } from '../hooks/useNotifications'
import { useTodaySchedule } from '../hooks/useTodaySchedule'
import { useTeacherData, useTeacherTodayLessons, formatDayHeader } from '../hooks/useTeacherData'
import { useCuratorStudents } from '../hooks/useCuratorStudents'
import { LessonCard } from '../components/LessonCard'
import { fetchGroups, fetchSubjects } from '../api/schedule'
import { useQuery } from '@tanstack/react-query'
import type { Lesson } from '../types/api'

/**
 * Домашняя страница — Dashboard с контентом по роли (Phase 13.5+/demo):
 *   student → сегодня + непрочитанные + PWA-подсказка
 *   teacher → мои предметы/группы + расписание по teacher_id (когда доступно)
 *   curator → студенты моей группы + сегодня (групповое расписание)
 *   admin   → сводка платформы
 * Неавторизованный пользователь сюда не попадает (ProtectedRoute).
 */
export function HomePage() {
  const { user } = useAuth()

  return (
    <section className="flex flex-col gap-4">
      <h1 className="text-lg font-semibold text-text">
        {user ? `Здравствуйте, ${user.full_name}!` : 'Здравствуйте!'}
      </h1>

      {user?.role === 'student' && <StudentDashboard />}
      {user?.role === 'teacher' && <TeacherDashboard />}
      {user?.role === 'curator' && <CuratorDashboard />}
      {user?.role === 'admin' && <AdminDashboard />}

      <p className="mt-2 text-center text-[11px] text-muted">
        QADAM · приложение можно установить на телефон: меню браузера → «Установить»
      </p>
    </section>
  )
}

// ---------- общие блоки ----------

function UnreadBanner() {
  const { unreadCount } = useNotifications()
  if (unreadCount <= 0) return null
  return (
    <a
      href="/notifications"
      className="rounded-lg border border-primary bg-primary/10 p-3 text-sm text-primary"
    >
      🔔 Непрочитанных уведомлений: {unreadCount} — открыть
    </a>
  )
}

function TodayLessonsCard({ lessons, isSunday, displayDate }: {
  lessons: Lesson[]
  isSunday: boolean
  displayDate: Date
}) {
  return (
    <div className="rounded-lg border border-surface bg-surface p-4">
      <h2 className="mb-2 text-sm font-semibold text-text">
        {isSunday ? 'Завтра (понедельник)' : 'Сегодня'}{' '}
        <span className="font-normal text-muted">· {formatDayHeader(displayDate)}</span>
      </h2>
      {lessons.length === 0 ? (
        <p className="py-2 text-sm text-muted">Сегодня занятий нет 🎉</p>
      ) : (
        <div className="flex flex-col gap-2">
          {lessons.map((lesson) => (
            <LessonCard key={lesson.template_id + lesson.date} lesson={lesson} />
          ))}
        </div>
      )}
    </div>
  )
}

function DashboardError({ what }: { what: string }) {
  return (
    <p role="alert" className="rounded-lg bg-warning/10 px-3 py-2 text-sm text-warning">
      Не удалось загрузить {what}. Попробуйте позже.
    </p>
  )
}

// ---------- student ----------

function StudentDashboard() {
  // Расписание студента строится по группе; выбор группы пока вручную в
  // /schedule — здесь берём первую группу как демо-умолчание (см. TODO ниже).
  const { data: groups } = useGroupsIfStudent()
  const groupId = groups?.[0]?.id
  const today = useTodaySchedule(groupId ? { groupId } : null)

  // TODO(student-flow): персональная группа придёт через /users/me → student
  // profile (Phase 9 UI), пока демо: первая группа из списка.

  return (
    <>
      <UnreadBanner />
      {today.error && <DashboardError what="расписание на сегодня" />}
      {today.isLoading && <p className="text-sm text-muted">Загрузка расписания…</p>}
      {today.enabled && !today.isLoading && (
        <TodayLessonsCard
          lessons={today.lessons}
          isSunday={today.isSunday}
          displayDate={today.displayDate}
        />
      )}
      <QuickLinks
        links={[
          { to: '/schedule', label: 'Полное расписание', icon: '📅' },
          { to: '/notifications', label: 'Уведомления', icon: '🔔' },
        ]}
      />
    </>
  )
}

function useGroupsIfStudent() {
  return useQuery({
    queryKey: ['groups'],
    queryFn: fetchGroups,
    select: (data) => data.groups,
    staleTime: 5 * 60_000,
    retry: 1,
  })
}

// ---------- teacher ----------

function TeacherDashboard() {
  const { user } = useAuth()
  const teacher = useTeacherData(user?.role)

  // Расписание преподавателя требует teacher_id (UUID из teachers), а не
  // user_id. Связка будет доступна через /teachers/me (16.7 backend) —
  // пока показываем заглушку с кнопкой на расписание.
  const teacherId = null
  const today = useTeacherTodayLessons(teacherId)

  return (
    <>
      <UnreadBanner />

      <div className="rounded-lg border border-surface bg-surface p-4">
        <h2 className="mb-2 text-sm font-semibold text-text">Мои предметы</h2>
        {teacher.subjectsLoading && <p className="text-sm text-muted">Загрузка…</p>}
        {teacher.subjectsError && <DashboardError what="предметы" />}
        {!teacher.subjectsLoading && (
          <div className="flex flex-wrap gap-2">
            {teacher.subjects.map((s) => (
              <span
                key={s.id}
                className="rounded-full bg-secondary/15 px-3 py-1 text-xs text-secondary"
              >
                {s.name}
              </span>
            ))}
          </div>
        )}
      </div>

      <div className="rounded-lg border border-surface bg-surface p-4">
        <h2 className="mb-2 text-sm font-semibold text-text">Мои группы</h2>
        {teacher.groupsLoading && <p className="text-sm text-muted">Загрузка…</p>}
        {teacher.groupsError && <DashboardError what="группы" />}
        {!teacher.groupsLoading && (
          <div className="flex flex-wrap gap-2">
            {teacher.groups.map((g) => (
              <span key={g.id} className="rounded-full bg-primary/10 px-3 py-1 text-xs text-primary">
                {g.name}
              </span>
            ))}
          </div>
        )}
      </div>

      {/* Сегодняшние занятия преподавателя — появятся с /teachers/me (Phase 16.7+) */}
      {today.data && today.data.length > 0 && (
        <TodayLessonsCard lessons={today.data} isSunday={false} displayDate={new Date()} />
      )}

      <QuickLinks
        links={[
          { to: '/schedule', label: 'Расписание (выберите группу)', icon: '📅' },
          { to: '/notifications', label: 'Уведомления', icon: '🔔' },
        ]}
      />
    </>
  )
}

// ---------- curator ----------

function CuratorDashboard() {
  const { data: groups } = useGroupsIfStudent()
  // Демо-умолчание: первая группа (привязка куратор→группа придёт через
  // /users/me → curator profile; сейчас groups.curator_id известен только backend'у).
  const groupId = groups?.[0]?.id
  const students = useCuratorStudents(groupId)
  const today = useTodaySchedule(groupId ? { groupId } : null)

  return (
    <>
      <UnreadBanner />

      <div className="rounded-lg border border-surface bg-surface p-4">
        <h2 className="mb-2 text-sm font-semibold text-text">Моя группа</h2>
        {students.isLoading && <p className="text-sm text-muted">Загрузка студентов…</p>}
        {students.error && <DashboardError what="студенты группы" />}
        {students.data && (
          <>
            <p className="text-sm text-muted">
              Студентов: <span className="font-semibold text-text">{students.data.length}</span>
            </p>
            <ul className="mt-2 flex flex-col gap-1">
              {students.data.map((s) => (
                <li key={s.id} className="flex items-center justify-between text-sm">
                  <span className="text-text">{s.full_name ?? '—'}</span>
                  <span className={`text-xs ${s.status === 'active' ? 'text-primary' : 'text-warning'}`}>
                    {statusLabel(s.status)}
                  </span>
                </li>
              ))}
            </ul>
          </>
        )}
      </div>

      {today.error && <DashboardError what="расписание группы" />}
      {today.enabled && !today.isLoading && (
        <TodayLessonsCard
          lessons={today.lessons}
          isSunday={today.isSunday}
          displayDate={today.displayDate}
        />
      )}

      <QuickLinks
        links={[
          { to: '/schedule', label: 'Расписание группы', icon: '📅' },
          { to: '/notifications', label: 'Уведомления', icon: '🔔' },
        ]}
      />
    </>
  )
}

function statusLabel(status: string): string {
  switch (status) {
    case 'active':
      return 'активен'
    case 'expelled':
      return 'отчислен'
    case 'academic_leave':
      return 'академ. отпуск'
    case 'graduated':
      return 'выпущен'
    default:
      return status
  }
}

// ---------- admin ----------

function AdminDashboard() {
  const groups = useGroupsIfStudent()
  const subjects = useQuery({
    queryKey: ['subjects'],
    queryFn: fetchSubjects,
    staleTime: 5 * 60_000,
    retry: 1,
  })

  return (
    <>
      <UnreadBanner />

      <div className="grid grid-cols-2 gap-3">
        <StatCard label="Группы" value={groups.data?.length ?? '…'} />
        <StatCard label="Предметы" value={subjects.data?.length ?? '…'} />
      </div>

      <div className="rounded-lg border border-surface bg-surface p-4 text-sm text-muted">
        <p className="font-medium text-text">Панель администратора</p>
        <p className="mt-1">
          Управление пользователями, расписанием и заменами — через API
          (Admin Panel UI в разработке). Аудит-лог: <code className="text-primary">/admin/audit-logs</code>.
        </p>
      </div>

      <QuickLinks
        links={[
          { to: '/schedule', label: 'Расписание групп', icon: '📅' },
          { to: '/notifications', label: 'Уведомления', icon: '🔔' },
        ]}
      />
    </>
  )
}

function StatCard({ label, value }: { label: string; value: number | string }) {
  return (
    <div className="rounded-lg border border-surface bg-surface p-4 text-center">
      <p className="text-2xl font-bold text-primary">{value}</p>
      <p className="text-xs text-muted">{label}</p>
    </div>
  )
}

// ---------- общие ----------

function QuickLinks({ links }: { links: { to: string; label: string; icon: string }[] }) {
  return (
    <div className="grid grid-cols-2 gap-3">
      {links.map((l) => (
        <a
          key={l.to}
          href={l.to}
          className="rounded-lg border border-surface bg-surface p-3 text-center text-xs text-text transition-colors hover:border-primary"
        >
          <span aria-hidden="true" className="mb-1 block text-xl">{l.icon}</span>
          {l.label}
        </a>
      ))}
    </div>
  )
}

// Фикс линтера для неиспользуемых импортов убран — fetchGroups/fetchMaterialsBySubject используются ниже.
