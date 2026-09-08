package locales

import (
	"testing"
)

func TestNormalize(t *testing.T) {
	cases := []struct {
		input string
		want  Language
	}{
		{"kz", Kazakh},
		{"ru", Russian},
		{"en", English},
		{"", Russian},     // пустой — fallback
		{"de", Russian},   // неизвестный — fallback
		{"RUSSIAN", Russian}, // регистр не поддерживается осознанно (коды в нижнем регистре)
	}
	for _, c := range cases {
		if got := Normalize(c.input); got != c.want {
			t.Errorf("Normalize(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestParse(t *testing.T) {
	// валидные
	for _, lang := range []string{"kz", "ru", "en"} {
		if _, err := Parse(lang); err != nil {
			t.Errorf("Parse(%q) returned error: %v", lang, err)
		}
	}
	// невалидные
	for _, lang := range []string{"", "de", "kazakh"} {
		if _, err := Parse(lang); err == nil {
			t.Errorf("Parse(%q) expected error, got nil", lang)
		}
	}
}

func TestGet_FlattensNestedKeys(t *testing.T) {
	dict := Get("ru")

	// иерархический ключ развёрнут в плоский
	if v, ok := dict["schedule.today"]; !ok || v != "Сегодня" {
		t.Errorf("expected schedule.today=Сегодня, got %q (ok=%v)", v, ok)
	}
	if v, ok := dict["notifications.empty_state"]; !ok || v != "Уведомлений нет" {
		t.Errorf("expected notifications.empty_state, got %q (ok=%v)", v, ok)
	}
	// вложенные категории материалов
	if v, ok := dict["materials.categories.lecture"]; !ok || v != "Лекции" {
		t.Errorf("expected materials.categories.lecture=Лекции, got %q (ok=%v)", v, ok)
	}
}

func TestGet_AllLanguagesSameKeys(t *testing.T) {
	// Все три словаря должны иметь одинаковый набор ключей — иначе
	// интерфейс на одном языке потеряет часть текстов.
	reference := Get("ru")
	for _, lang := range []Language{Kazakh, English} {
		dict := Get(lang.String())
		if len(dict) != len(reference) {
			t.Errorf("language %q has %d keys, reference (ru) has %d", lang, len(dict), len(reference))
		}
		for key := range reference {
			if _, ok := dict[key]; !ok {
				t.Errorf("language %q missing key %q", lang, key)
			}
		}
	}
}

func TestTranslate(t *testing.T) {
	// существующий ключ
	if got := Translate("ru", "common.save"); got != "Сохранить" {
		t.Errorf("expected Сохранить, got %q", got)
	}
	// fallback: ключ есть в ru, но запрошен язык — ключ есть во всех, поэтому
	// проверяем fallback через несуществующий язык (нормализуется к ru)
	if got := Translate("de", "common.save"); got != "Сохранить" {
		t.Errorf("expected fallback to ru, got %q", got)
	}
	// отсутствующий ключ — возвращается сам ключ
	if got := Translate("ru", "nonexistent.key"); got != "nonexistent.key" {
		t.Errorf("expected key as fallback, got %q", got)
	}
}
