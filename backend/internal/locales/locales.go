// Package locales предоставляет словари переводов интерфейса (раздел 26
// спецификации): Қазақша (kz), Русский (ru), English (en). Словари хранятся
// в JSON-файлах рядом с пакетом и встраиваются в бинарник через go:embed.
// Ключи переводов иерархические ("schedule.today"), fallback — русский.
package locales

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed kz.json ru.json en.json
var localeFS embed.FS

// Language — код поддерживаемого языка интерфейса.
type Language string

const (
	Kazakh  Language = "kz"
	Russian Language = "ru"
	English Language = "en"
)

// DefaultLanguage — язык по умолчанию (fallback, раздел 26: "язык по
// умолчанию — определяется профилем пользователя, с fallback на русский").
const DefaultLanguage = Russian

// SupportedLanguages — все поддерживаемые языки.
var SupportedLanguages = []Language{Kazakh, Russian, English}

// isValid проверяет, поддерживается ли язык.
func (l Language) isValid() bool {
	for _, supported := range SupportedLanguages {
		if l == supported {
			return true
		}
	}
	return false
}

// Normalize приводит код языка к поддерживаемому: неизвестный/пустой
// заменяется на DefaultLanguage (fallback на русский).
func Normalize(lang string) Language {
	l := Language(lang)
	if l.isValid() {
		return l
	}
	return DefaultLanguage
}

// Parse проверяет код языка и возвращает ошибку для неподдерживаемого —
// используется при валидации пользовательского ввода (смена языка в профиле).
func Parse(lang string) (Language, error) {
	l := Language(lang)
	if !l.isValid() {
		return "", fmt.Errorf("unsupported language %q (supported: kz, ru, en)", lang)
	}
	return l, nil
}

// Dictionary — плоская карта "иерархический ключ → перевод",
// например {"schedule.today": "Сегодня"}.
type Dictionary map[string]string

// dictionaries — загруженные при старте словари по языкам.
var dictionaries map[Language]Dictionary

func init() {
	dictionaries = make(map[Language]Dictionary, len(SupportedLanguages))
	for _, lang := range SupportedLanguages {
		data, err := localeFS.ReadFile(lang.String() + ".json")
		if err != nil {
			panic(fmt.Sprintf("locales: embedded file %s.json not readable: %v", lang, err))
		}
		dict, err := flatten(data)
		if err != nil {
			panic(fmt.Sprintf("locales: failed to parse %s.json: %v", lang, err))
		}
		dictionaries[lang] = dict
	}
}

func (l Language) String() string { return string(l) }

// flatten разворачивает вложенный JSON в плоскую карту с точечной нотацией
// ключей ("schedule": {"today": "..."} → "schedule.today": "...").
func flatten(data []byte) (Dictionary, error) {
	var nested map[string]any
	if err := json.Unmarshal(data, &nested); err != nil {
		return nil, err
	}

	dict := Dictionary{}
	var walk func(prefix string, node map[string]any)
	walk = func(prefix string, node map[string]any) {
		for key, value := range node {
			fullKey := key
			if prefix != "" {
				fullKey = prefix + "." + key
			}
			switch v := value.(type) {
			case string:
				dict[fullKey] = v
			case map[string]any:
				walk(fullKey, v)
			default:
				// нестроковые значения (числа и пр.) игнорируются
			}
		}
	}
	walk("", nested)
	return dict, nil
}

// Get возвращает словарь переводов для языка. Неизвестный язык
// нормализуется к DefaultLanguage.
func Get(lang string) Dictionary {
	return dictionaries[Normalize(lang)]
}

// Translate возвращает перевод по ключу с fallback на DefaultLanguage,
// затем на сам ключ (чтобы отсутствие перевода было видно, а не молча
// возвращало пустую строку).
func Translate(lang, key string) string {
	if v, ok := dictionaries[Normalize(lang)][key]; ok {
		return v
	}
	if v, ok := dictionaries[DefaultLanguage][key]; ok {
		return v
	}
	return key
}
