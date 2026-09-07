-- Phase 7 (Materials): materials + material_files — учебные материалы,
-- привязанные к предметам (разделы 14, 21, 34.1 спецификации).
--
-- Файлы НЕ хранятся в PostgreSQL как BLOB — только метаданные; сами файлы
-- лежат во внешнем хранилище через абстракцию FileStorage (раздел 14).

CREATE TABLE materials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id UUID NOT NULL REFERENCES subjects(id),
    category VARCHAR(20) NOT NULL DEFAULT 'extra'
        CHECK (category IN ('lecture', 'practice', 'lab', 'extra')),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_materials_subject ON materials(subject_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_materials_created_by ON materials(created_by) WHERE deleted_at IS NULL;

CREATE TABLE material_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    material_id UUID NOT NULL REFERENCES materials(id) ON DELETE CASCADE,
    file_type VARCHAR(20) NOT NULL
        CHECK (file_type IN ('pdf', 'docx', 'pptx', 'image', 'video_link', 'link')),
    storage_key VARCHAR(512),
    external_url VARCHAR(2048),
    file_name VARCHAR(255) NOT NULL,
    size_bytes BIGINT,
    mime_type VARCHAR(255),
    uploaded_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Файл хранится в S3 (storage_key) ИЛИ это внешняя ссылка (external_url);
    -- file_type 'video_link'/'link' требуют external_url, остальные — storage_key.
    CHECK (
        (file_type IN ('video_link', 'link') AND external_url IS NOT NULL)
        OR
        (file_type NOT IN ('video_link', 'link') AND storage_key IS NOT NULL)
    )
);

CREATE INDEX idx_material_files_material ON material_files(material_id);

COMMENT ON TABLE materials IS 'Логические учебные материалы по предметам (Phase 7)';
COMMENT ON TABLE material_files IS 'Файлы/ссылки материалов: metadata только, содержимое — во внешнем хранилище (раздел 14)';
