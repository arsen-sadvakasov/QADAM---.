// Command seed-demo наполняет базу QADAM реалистичными демонстрационными
// данными (Phase: Demo Data). Идемпотентен: перед вставкой проверяет
// существование по имени/username и пропускает уже созданное — повторный
// запуск безопасен и ничего не удаляет и не перезаписывает.
//
// Использование:
//
//	cd backend && go run ./cmd/seed-demo
//
// Подключение берётся из DATABASE_URL (как у сервера). Существующий
// администратор не изменяется. Все данные вымышленные.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/qadam/backend/internal/config"
	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/services"
)

// demoPassword — единый пароль всех демо-пользователей (только для
// демонстрации; не является production-секретом и в Git не выносится).
const demoPassword = "demo12345"

func main() {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := config.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pool.Close()

	userRepo := repositories.NewUserRepository(pool)
	roleRepo := repositories.NewRoleRepository(pool)
	userAdmin := services.NewUserAdminService(userRepo, roleRepo)

	specialtyRepo := repositories.NewSpecialtyRepository(pool)
	courseRepo := repositories.NewCourseRepository(pool)
	groupRepo := repositories.NewGroupRepository(pool)
	subjectRepo := repositories.NewSubjectRepository(pool)
	roomRepo := repositories.NewRoomRepository(pool)
	teacherRepo := repositories.NewTeacherRepository(pool)
	studentRepo := repositories.NewStudentRepository(pool)
	scheduleRepo := repositories.NewScheduleTemplateRepository(pool)
	materialRepo := repositories.NewMaterialRepository(pool)
	notificationRepo := repositories.NewNotificationRepository(pool)

	// ---------- 1. Специальности ----------
	specialties := map[string]string{} // name -> id
	for _, name := range []string{
		"Программное обеспечение",
		"Информационные системы",
		"Кибербезопасность",
	} {
		if existing, err := findSpecialty(ctx, specialtyRepo, name); err == nil {
			specialties[name] = existing.ID
			continue
		}
		id, err := specialtyRepo.Create(ctx, &models.Specialty{Name: name})
		must(err, "create specialty "+name)
		specialties[name] = id
		fmt.Printf("✓ specialty %s\n", name)
	}

	// ---------- 2. Курсы ----------
	courses := map[string]string{} // "<specialty_id>:<year>" -> id
	mustCourse := func(specialtyID string, year int) string {
		key := fmt.Sprintf("%s:%d", specialtyID, year)
		if id, ok := courses[key]; ok {
			return id
		}
		for _, c := range mustListCourses(ctx, courseRepo, specialtyID) {
			if c.YearNumber == year {
				courses[key] = c.ID
				return c.ID
			}
		}
		id, err := courseRepo.Create(ctx, &models.Course{SpecialtyID: specialtyID, YearNumber: year})
		must(err, "create course")
		courses[key] = id
		fmt.Printf("✓ course %d год\n", year)
		return id
	}

	// ---------- 3. Группы ----------
	groups := map[string]string{} // name -> id
	type groupSpec struct {
		name          string
		specialtyName string
		year          int
		curatorID     *string
	}
	var groupSpecs []groupSpec // заполняется после создания кураторов
	groupNames := []string{"ПО-23", "ПО-24", "ИС-31", "КБ-32"}

	ensureGroups := func() {
		for _, gs := range groupSpecs {
			if id, ok := groups[gs.name]; ok {
				_ = id
				continue
			}
			if existing, err := groupRepo.FindByID(ctx, lookupGroupID(ctx, groupRepo, gs.name)); err == nil {
				groups[gs.name] = existing.ID
				continue
			}
			courseID := mustCourse(specialties[gs.specialtyName], gs.year)
			g := &models.Group{
				SpecialtyID: specialties[gs.specialtyName],
				CourseID:    courseID,
				CuratorID:   gs.curatorID,
				Name:        gs.name,
			}
			id, err := groupRepo.Create(ctx, g)
			must(err, "create group "+gs.name)
			groups[gs.name] = id
			fmt.Printf("✓ group %s (%s, %d курс)\n", gs.name, gs.specialtyName, gs.year)
		}
	}

	// ---------- 4. Предметы ----------
	subjects := map[string]string{} // name -> id
	for _, name := range []string{
		"Web-программирование",
		"Базы данных",
		"Программирование",
		"Алгоритмы и структуры данных",
		"Компьютерные сети",
		"Информационная безопасность",
		"Операционные системы",
		"Проектирование программного обеспечения",
		"Математика",
		"Английский язык",
	} {
		if existing := lookupSubjectID(ctx, subjectRepo, name); existing != "" {
			subjects[name] = existing
			continue
		}
		id, err := subjectRepo.Create(ctx, &models.Subject{Name: name})
		must(err, "create subject "+name)
		subjects[name] = id
		fmt.Printf("✓ subject %s\n", name)
	}

	// ---------- 5. Кабинеты ----------
	rooms := map[string]string{} // number -> id
	type roomSpec struct {
		number string
		name   string
		typ    models.RoomType
		floor  int
	}
	for _, r := range []roomSpec{
		{"101", "Лекционный зал", models.RoomTypeLecture, 1},
		{"205", "Компьютерный класс", models.RoomTypeComputer, 2},
		{"206", "Компьютерный класс", models.RoomTypeComputer, 2},
		{"301", "Лаборатория сетей", models.RoomTypeLab, 3},
		{"302", "Лаборатория ИБ", models.RoomTypeLab, 3},
		{"405", "Кабинет иностранных языков", models.RoomTypeOther, 4},
	} {
		if existing := lookupRoomID(ctx, roomRepo, r.number); existing != "" {
			rooms[r.number] = existing
			continue
		}
		floor := r.floor
		name := r.name
		id, err := roomRepo.Create(ctx, &models.Room{
			Number: r.number,
			Name:   &name,
			Type:   r.typ,
			Floor:  &floor,
		})
		must(err, "create room "+r.number)
		rooms[r.number] = id
		fmt.Printf("✓ room %s (%s)\n", r.number, r.name)
	}

	// ---------- 6. Пользователи: преподаватели ----------
	// createDemoUser возвращает (userID, created)
	createDemoUser := func(username, fullName, email, role string) (string, bool) {
		if u, err := userRepo.FindByUsername(ctx, username); err == nil {
			return u.ID, false // уже существует — не трогаем
		}
		u, err := userAdmin.Create(ctx, "", services.CreateUserInput{
			Username: username,
			Password: demoPassword,
			FullName: fullName,
			Role:     models.RoleKey(role),
			Email:    &email,
			Language: "ru",
			Theme:    "dark",
		})
		must(err, "create user "+username)
		return u.ID, true
	}

	type teacherSpec struct {
		username, fullName, email string
		teaches                   []string // названия предметов
	}
	teachersSpec := []teacherSpec{
		{"aidar.nurlanov", "Айдар Нурланов", "aidar.nurlanov@qadam.demo",
			[]string{"Программирование", "Алгоритмы и структуры данных"}},
		{"dana.seitova", "Дана Сейтова", "dana.seitova@qadam.demo",
			[]string{"Базы данных", "Проектирование программного обеспечения"}},
		{"marat.akhmetov", "Марат Ахметов", "marat.akhmetov@qadam.demo",
			[]string{"Web-программирование", "Компьютерные сети", "Операционные системы"}},
		{"gulnara.iskakova", "Гульнара Искакова", "gulnara.iskakova@qadam.demo",
			[]string{"Информационная безопасность"}},
		{"alen.sadvakasov", "Ален Садвакасов", "alen.sadvakasov@qadam.demo",
			[]string{"Математика", "Английский язык"}},
	}
	teachers := map[string]string{} // username -> teacherID
	for _, ts := range teachersSpec {
		userID, created := createDemoUser(ts.username, ts.fullName, ts.email, "teacher")
		if tid, err := lookupTeacherID(ctx, teacherRepo, userID); err == nil {
			teachers[ts.username] = tid
			continue
		}
		tid, err := teacherRepo.Create(ctx, userID, nil)
		must(err, "create teacher "+ts.username)
		teachers[ts.username] = tid
		if created {
			fmt.Printf("✓ teacher %s (%s)\n", ts.fullName, ts.username)
		}
	}

	// ---------- 7. Кураторы (пользователи с ролью curator) ----------
	curators := map[string]string{} // username -> userID
	curatorSpecs := []struct {
		username, fullName, email, groupName string
	}{
		{"aigul.muratova", "Айгуль Муратова", "aigul.muratova@qadam.demo", "ПО-23"},
		{"sergey.petrov", "Сергей Петров", "sergey.petrov@qadam.demo", "ИС-31"},
	}
	for _, cs := range curatorSpecs {
		userID, created := createDemoUser(cs.username, cs.fullName, cs.email, "curator")
		curators[cs.username] = userID
		if created {
			fmt.Printf("✓ curator %s (%s)\n", cs.fullName, cs.username)
		}
	}

	// ---------- 8. Группы (создание с кураторами) ----------
	groupSpecs = []groupSpec{
		{"ПО-23", "Программное обеспечение", 2, ptr(curators["aigul.muratova"])},
		{"ПО-24", "Программное обеспечение", 2, nil},
		{"ИС-31", "Информационные системы", 3, ptr(curators["sergey.petrov"])},
		{"КБ-32", "Кибербезопасность", 3, nil},
	}
	ensureGroups()

	// ---------- 9. Студенты ----------
	studentSpecs := []struct{ fullName, username, email, group string }{
		{"Арман Алиев", "student01", "student01@qadam.demo", "ПО-23"},
		{"Данияр Садыков", "student02", "student02@qadam.demo", "ПО-23"},
		{"Алина Ермекова", "student03", "student03@qadam.demo", "ПО-23"},
		{"Мадина Касымова", "student04", "student04@qadam.demo", "ПО-23"},
		{"Тимур Нурбеков", "student05", "student05@qadam.demo", "ПО-23"},
		{"Айша Тлеубаева", "student06", "student06@qadam.demo", "ПО-24"},
		{"Руслан Омаров", "student07", "student07@qadam.demo", "ПО-24"},
		{"Диана Сарсенова", "student08", "student08@qadam.demo", "ПО-24"},
		{"Ерасыл Беков", "student09", "student09@qadam.demo", "ПО-24"},
		{"Аружан Иманова", "student10", "student10@qadam.demo", "ПО-24"},
		{"Санжар Абдрахманов", "student11", "student11@qadam.demo", "ИС-31"},
		{"Назерке Жумабаева", "student12", "student12@qadam.demo", "ИС-31"},
		{"Ислам Конысов", "student13", "student13@qadam.demo", "ИС-31"},
		{"Жанна Ержанова", "student14", "student14@qadam.demo", "КБ-32"},
		{"Бекзат Турганов", "student15", "student15@qadam.demo", "КБ-32"},
	}
	studentsByGroup := map[string][]string{} // groupName -> userIDs
	createdStudents := 0
	for _, ss := range studentSpecs {
		userID, created := createDemoUser(ss.username, ss.fullName, ss.email, "student")
		studentsByGroup[ss.group] = append(studentsByGroup[ss.group], userID)
		if !created {
			continue
		}
		_, err := studentRepo.Create(ctx, &models.Student{
			UserID:  &userID,
			GroupID: groups[ss.group],
			Status:  models.StudentStatusActive,
		})
		must(err, "create student "+ss.username)
		createdStudents++
		fmt.Printf("✓ student %s → %s\n", ss.fullName, ss.group)
	}

	// ---------- 10. Назначения преподавателей на предметы (teacher_subjects) ----------
	type assignSpec struct {
		teacherUsername, subject string
		groupNames               []string
	}
	for _, a := range []assignSpec{
		{"aidar.nurlanov", "Программирование", []string{"ПО-23", "ПО-24"}},
		{"aidar.nurlanov", "Алгоритмы и структуры данных", []string{"ИС-31"}},
		{"dana.seitova", "Базы данных", []string{"ПО-23", "ПО-24", "ИС-31"}},
		{"dana.seitova", "Проектирование программного обеспечения", []string{"ИС-31"}},
		{"marat.akhmetov", "Web-программирование", []string{"ПО-23", "ПО-24"}},
		{"marat.akhmetov", "Компьютерные сети", []string{"ИС-31", "КБ-32"}},
		{"marat.akhmetov", "Операционные системы", []string{"ПО-24"}},
		{"gulnara.iskakova", "Информационная безопасность", []string{"ИС-31", "КБ-32"}},
		{"alen.sadvakasov", "Математика", []string{"ПО-23", "ПО-24"}},
		{"alen.sadvakasov", "Английский язык", []string{"ПО-23", "ИС-31", "КБ-32"}},
	} {
		for _, gn := range a.groupNames {
			ts := models.TeacherSubject{
				TeacherID: teachers[a.teacherUsername],
				SubjectID: subjects[a.subject],
				GroupID:   groups[gn],
			}
			// Идемпотентность: уникальный индекс в БД; при повторе — пропускаем.
			if err := teacherRepo.AssignSubject(ctx, ts); err != nil {
				fmt.Printf("  = assignment %s/%s/%s already exists\n", a.teacherUsername, a.subject, gn)
				continue
			}
			fmt.Printf("✓ assignment %s → %s (%s)\n", a.teacherUsername, a.subject, gn)
		}
	}

	// ---------- 11. Расписание (по шаблонам на неделю) ----------
	year := time.Now().Year()
	validFrom := time.Date(year, time.September, 1, 0, 0, 0, 0, time.UTC)
	clock := func(hh, mm int) time.Time {
		return time.Date(2000, 1, 1, hh, mm, 0, 0, time.UTC)
	}
	type slot struct {
		day        int // 1..5
		start, end [2]int
		subject    string
		teacher    string
		room       string
		lessonType models.LessonType
	}
	weekPlan := [][]slot{
		{ // Понедельник
			{1, [2]int{9, 0}, [2]int{10, 30}, "Программирование", "aidar.nurlanov", "205", models.LessonTypePractice},
			{1, [2]int{10, 45}, [2]int{12, 15}, "Базы данных", "dana.seitova", "206", models.LessonTypeLecture},
			{1, [2]int{12, 30}, [2]int{14, 0}, "Английский язык", "alen.sadvakasov", "405", models.LessonTypeSeminar},
		},
		{ // Вторник
			{2, [2]int{9, 0}, [2]int{10, 30}, "Web-программирование", "marat.akhmetov", "206", models.LessonTypeLab},
			{2, [2]int{10, 45}, [2]int{12, 15}, "Математика", "alen.sadvakasov", "101", models.LessonTypeLecture},
			{2, [2]int{12, 30}, [2]int{14, 0}, "Компьютерные сети", "marat.akhmetov", "301", models.LessonTypeLecture},
		},
		{ // Среда
			{3, [2]int{9, 0}, [2]int{10, 30}, "Базы данных", "dana.seitova", "206", models.LessonTypePractice},
			{3, [2]int{10, 45}, [2]int{12, 15}, "Алгоритмы и структуры данных", "aidar.nurlanov", "205", models.LessonTypeLecture},
			{3, [2]int{12, 30}, [2]int{14, 0}, "Информационная безопасность", "gulnara.iskakova", "302", models.LessonTypeLecture},
		},
		{ // Четверг
			{4, [2]int{9, 0}, [2]int{10, 30}, "Web-программирование", "marat.akhmetov", "206", models.LessonTypePractice},
			{4, [2]int{10, 45}, [2]int{12, 15}, "Операционные системы", "marat.akhmetov", "301", models.LessonTypeLecture},
			{4, [2]int{12, 30}, [2]int{14, 0}, "Английский язык", "alen.sadvakasov", "405", models.LessonTypeSeminar},
		},
		{ // Пятница
			{5, [2]int{9, 0}, [2]int{10, 30}, "Проектирование программного обеспечения", "dana.seitova", "101", models.LessonTypeLecture},
			{5, [2]int{10, 45}, [2]int{12, 15}, "Программирование", "aidar.nurlanov", "205", models.LessonTypeLab},
			{5, [2]int{12, 30}, [2]int{14, 0}, "Информационная безопасность", "gulnara.iskakova", "302", models.LessonTypeLecture},
		},
	}

	createdSchedule := 0
	for _, gn := range groupNames {
		for _, day := range weekPlan {
			for _, s := range day {
				// Идемпотентность: пропускаем слот, если для группы на этот день
				// и время начала уже есть занятие.
				if scheduleSlotExists(ctx, scheduleRepo, groups[gn], s.day, clock(s.start[0], s.start[1])) {
					continue
				}
				t := &models.ScheduleTemplate{
					GroupID:    groups[gn],
					SubjectID:  subjects[s.subject],
					TeacherID:  teachers[s.teacher],
					RoomID:     rooms[s.room],
					DayOfWeek:  s.day,
					StartTime:  clock(s.start[0], s.start[1]),
					EndTime:    clock(s.end[0], s.end[1]),
					LessonType: s.lessonType,
					WeekParity: models.WeekParityAll,
					ValidFrom:  validFrom,
					Status:     models.ScheduleTemplateStatusActive,
				}
				if _, err := scheduleRepo.Create(ctx, t); err != nil {
					// Конфликт (пересечение по преподавателю/кабинету) — для ПО-24
					// сдвигаем кабинет на соседний и пробуем ещё раз.
					t.RoomID = rooms["206"]
					if t.RoomID == rooms[s.room] {
						t.RoomID = rooms["205"]
					}
					if _, err2 := scheduleRepo.Create(ctx, t); err2 != nil {
						fmt.Printf("  = skip slot (conflict): %s %s %s %s\n", gn, dayName(s.day), s.subject, clock(s.start[0], s.start[1]).Format("15:04"))
						continue
					}
				}
				createdSchedule++
			}
		}
		fmt.Printf("✓ schedule for %s (%d slots)\n", gn, len(weekPlan)*3)
	}
	_ = createdSchedule

	// ---------- 12. Материалы ----------
	createdMaterials := 0
	type materialSpec struct {
		subject, category, title, description, author string
	}
	for _, m := range []materialSpec{
		{"Web-программирование", "lecture", "Основы HTML", "Структура документа, семантические теги, формы.", "marat.akhmetov"},
		{"Web-программирование", "lecture", "CSS: Flexbox и Grid", "Современные системы раскладки на CSS.", "marat.akhmetov"},
		{"Web-программирование", "practice", "JavaScript: основы", "Типы, функции, массивы, объекты, DOM.", "marat.akhmetov"},
		{"Web-программирование", "extra", "React: компоненты", "Функциональные компоненты, props, состояние.", "marat.akhmetov"},
		{"Базы данных", "lecture", "Основы SQL", "DDL, DML, типы данных PostgreSQL.", "dana.seitova"},
		{"Базы данных", "practice", "SELECT и WHERE", "Выборка, фильтрация, сортировка, агрегаты.", "dana.seitova"},
		{"Базы данных", "practice", "JOIN", "INNER/LEFT/RIGHT JOIN, связи таблиц.", "dana.seitova"},
		{"Базы данных", "extra", "Проектирование базы данных", "Нормализация, ER-диаграммы, ключи.", "dana.seitova"},
		{"Информационная безопасность", "lecture", "Основы информационной безопасности", "Триада CIA, угрозы, модель нарушителя.", "gulnara.iskakova"},
		{"Информационная безопасность", "lecture", "Виды киберугроз", "Фишинг, малварь, DDoS, социальная инженерия.", "gulnara.iskakova"},
		{"Информационная безопасность", "practice", "Пароли и аутентификация", "Хэширование паролей, MFA, менеджеры паролей.", "gulnara.iskakova"},
		{"Информационная безопасность", "extra", "Основы защиты данных", "Шифрование, backup, персональные данные.", "gulnara.iskakova"},
	} {
		// Идемпотентность по title.
		if materialExists(ctx, materialRepo, subjects[m.subject], m.title) {
			continue
		}
		author, err := userRepo.FindByUsername(ctx, m.author)
		if err != nil {
			fmt.Printf("  = skip material %q: author %s not found\n", m.title, m.author)
			continue
		}
		desc := m.description
		_, err = materialRepo.Create(ctx, &models.Material{
			SubjectID:   subjects[m.subject],
			Category:    models.MaterialCategory(m.category),
			Title:       m.title,
			Description: &desc,
			CreatedBy:   author.ID,
		})
		if err != nil {
			fmt.Printf("  = skip material %q: %v\n", m.title, err)
			continue
		}
		createdMaterials++
		fmt.Printf("✓ material %s — %s\n", m.subject, m.title)
	}
	_ = createdMaterials

	// ---------- 13. Уведомления ----------
	createdNotifications := 0
	notify := func(nType models.NotificationType, title, body string, recipients []string) {
		// Идемпотентность: пропускаем, если хоть один получатель уже имеет
		// уведомление с таким заголовком (уникального индекса на title нет).
		if len(recipients) > 0 {
			existing, err := notificationRepo.ListForUser(ctx, recipients[0], nil, 100)
			if err == nil {
				for _, un := range existing {
					if un.Title == title {
						return
					}
				}
			}
		}
		entityType := "system"
		_, err := notificationRepo.Create(ctx, &models.Notification{
			Type:              nType,
			Title:             title,
			Body:              body,
			Priority:          models.NotificationPriorityNormal,
			RelatedEntityType: &entityType,
		}, recipients)
		if err != nil {
			fmt.Printf("  = skip notification %q: %v\n", title, err)
			return
		}
		createdNotifications++
		fmt.Printf("✓ notification %s (%d получателей)\n", title, len(recipients))
	}

	var allStudentIDs []string
	for _, ids := range studentsByGroup {
		allStudentIDs = append(allStudentIDs, ids...)
	}

	notify(models.NotificationSystem,
		"Расписание на новую неделю опубликовано",
		"Актуальное расписание занятий доступно в разделе «Расписание».",
		allStudentIDs)
	notify(models.NotificationNewMaterial,
		"Добавлен новый учебный материал",
		"По предмету «Web-программирование» опубликованы новые материалы.",
		studentsByGroup["ПО-23"])
	notify(models.NotificationScheduleChange,
		"Завтра занятие по базам данных в 09:00",
		"Не забудьте подготовиться к практическому занятию.",
		studentsByGroup["ПО-24"])
	notify(models.NotificationReplacement,
		"Изменено время занятия по Web-программированию",
		"Занятие перенесено: теперь оно начинается в 09:00.",
		studentsByGroup["ПО-23"])
	notify(models.NotificationNewMaterial,
		"Новый материал по информационной безопасности",
		"Опубликованы материалы по основам защиты данных.",
		append(studentsByGroup["ИС-31"], studentsByGroup["КБ-32"]...))

	// ---------- Итог ----------
	fmt.Println()
	fmt.Println("Demo data seeding complete!")
	fmt.Printf("Specialties: %d, Groups: %d, Subjects: %d, Rooms: %d\n",
		len(specialties), len(groups), len(subjects), len(rooms))
	fmt.Printf("Teachers: %d, Curators: %d, Students created now: %d (of %d)\n",
		len(teachers), len(curators), createdStudents, len(studentSpecs))
	fmt.Printf("Schedule slots created: %d, Materials: %d, Notifications: %d\n",
		createdSchedule, createdMaterials, createdNotifications)
	fmt.Printf("Demo password for all demo users: %s\n", demoPassword)
}

// ---------- helpers ----------

// scheduleSlotExists проверяет, есть ли уже занятие у группы в этот день
// с этим временем начала (идемпотентность seed'а).
func scheduleSlotExists(ctx context.Context, repo repositories.ScheduleTemplateRepository, groupID string, day int, start time.Time) bool {
	templates, err := repo.ListByGroup(ctx, groupID)
	if err != nil {
		return false
	}
	for _, t := range templates {
		if t.DayOfWeek == day && timeOfDayMinutes(t.StartTime) == timeOfDayMinutes(start) {
			return true
		}
	}
	return false
}

func timeOfDayMinutes(t time.Time) int {
	return t.Hour()*60 + t.Minute()
}

func must(err error, what string) {
	if err != nil {
		log.Fatalf("seed-demo: %s: %v", what, err)
	}
}

func ptr(s string) *string { return &s }

func dayName(isoDay int) string {
	names := []string{"", "Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота", "Воскресенье"}
	return names[isoDay]
}

func findSpecialty(ctx context.Context, repo repositories.SpecialtyRepository, name string) (*models.Specialty, error) {
	list, err := repo.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, s := range list {
		if s.Name == name {
			return s, nil
		}
	}
	return nil, repositories.ErrNotFound
}

func mustListCourses(ctx context.Context, repo repositories.CourseRepository, specialtyID string) []*models.Course {
	courses, err := repo.ListBySpecialty(ctx, specialtyID)
	must(err, "list courses")
	return courses
}

// lookupGroupID ищет группу по имени через List (FindByName в API-плане нет).
func lookupGroupID(ctx context.Context, repo repositories.GroupRepository, name string) string {
	groups, err := repo.List(ctx)
	if err != nil {
		return ""
	}
	for _, g := range groups {
		if g.Name == name {
			return g.ID
		}
	}
	return ""
}

func lookupSubjectID(ctx context.Context, repo repositories.SubjectRepository, name string) string {
	list, err := repo.List(ctx)
	if err != nil {
		return ""
	}
	for _, s := range list {
		if s.Name == name {
			return s.ID
		}
	}
	return ""
}

func lookupRoomID(ctx context.Context, repo repositories.RoomRepository, number string) string {
	list, err := repo.List(ctx)
	if err != nil {
		return ""
	}
	for _, r := range list {
		if r.Number == number {
			return r.ID
		}
	}
	return ""
}

// lookupTeacherID ищет запись teacher по связанному userID через List.
func lookupTeacherID(ctx context.Context, repo repositories.TeacherRepository, userID string) (string, error) {
	list, err := repo.List(ctx)
	if err != nil {
		return "", err
	}
	for _, t := range list {
		if t.UserID == userID {
			return t.ID, nil
		}
	}
	return "", repositories.ErrNotFound
}

func materialExists(ctx context.Context, repo repositories.MaterialRepository, subjectID, title string) bool {
	list, err := repo.ListBySubject(ctx, subjectID)
	if err != nil {
		return false
	}
	for _, m := range list {
		if m.Title == title {
			return true
		}
	}
	return false
}
