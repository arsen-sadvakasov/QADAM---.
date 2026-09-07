-- Phase 3 (Database Core): academic structure entities per spec section 34.1
-- (specialties, courses, groups, students, teachers, subjects, rooms), plus
-- supporting reference/join tables (academic_statuses, teacher_subjects).

CREATE TABLE academic_statuses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0
);

INSERT INTO academic_statuses (key, name, sort_order) VALUES
    ('excellent', 'Отличник', 1),
    ('good', 'Хорошист', 2),
    ('average', 'Успевающий', 3);

CREATE TABLE specialties (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    college_id UUID REFERENCES colleges(id),
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE courses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    specialty_id UUID NOT NULL REFERENCES specialties(id) ON DELETE CASCADE,
    year_number INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_courses_specialty_id ON courses (specialty_id);

CREATE TABLE groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    college_id UUID REFERENCES colleges(id),
    specialty_id UUID NOT NULL REFERENCES specialties(id),
    course_id UUID NOT NULL REFERENCES courses(id),
    curator_id UUID REFERENCES users(id),
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_groups_specialty_id ON groups (specialty_id);
CREATE INDEX idx_groups_course_id ON groups (course_id);
CREATE INDEX idx_groups_curator_id ON groups (curator_id);

CREATE TABLE subjects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    college_id UUID REFERENCES colleges(id),
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    college_id UUID REFERENCES colleges(id),
    number VARCHAR(50) NOT NULL,
    name VARCHAR(100),
    type VARCHAR(20) NOT NULL DEFAULT 'other'
        CHECK (type IN ('lecture', 'lab', 'computer', 'other')),
    floor INT,
    building VARCHAR(100),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE teachers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    college_id UUID REFERENCES colleges(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

-- Преподаватель ведёт конкретный предмет в конкретной группе (решает edge
-- case "один предмет — разные преподаватели у разных групп" — раздел 34.1).
CREATE TABLE teacher_subjects (
    teacher_id UUID NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    PRIMARY KEY (teacher_id, subject_id, group_id)
);

CREATE INDEX idx_teacher_subjects_subject_id ON teacher_subjects (subject_id);
CREATE INDEX idx_teacher_subjects_group_id ON teacher_subjects (group_id);

CREATE TABLE students (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE REFERENCES users(id) ON DELETE SET NULL,
    group_id UUID NOT NULL REFERENCES groups(id),
    status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'expelled', 'academic_leave', 'graduated')),
    academic_status_id UUID REFERENCES academic_statuses(id),
    scholarship_status VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

-- Индекс для быстрой выборки студентов группы (раздел 34.2).
CREATE INDEX idx_students_group_id ON students (group_id);
