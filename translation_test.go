package i18n

import (
	"testing"
)

const (
	Validity = "test_data/validity/"
	Count    = "test_data/count/"
)

func TestLanguage(t *testing.T) {
	fn := func(code string, expected bool) func(t *testing.T) {
		return func(t *testing.T) {
			lang := Language(code)
			if lang.Valid() != expected {
				t.Fail()
			}
		}
	}

	t.Run("empty", fn("", false))
	t.Run("too long", fn("een", false))
	t.Run("too short", fn("k", false))
	t.Run("non alpha", fn("43", false))
	t.Run("part alpha", fn("d3", false))
	t.Run("valid", fn("de", true))
}

func TestKey(t *testing.T) {
	fn := func(keys []string, expected string) func(t *testing.T) {
		return func(t *testing.T) {
			var rootKey Key
			for _, key := range keys {
				rootKey = rootKey.Append(key)
			}

			if rootKey.String() != expected {
				t.Fail()
			}
		}
	}

	t.Run("empty", fn([]string{}, ""))
	t.Run("empty level", fn([]string{""}, ""))
	t.Run("one level", fn([]string{"a"}, "a"))
	t.Run("two levels", fn([]string{"a", "b"}, "a.b"))
	t.Run("three levels", fn([]string{"a", "b", "c"}, "a.b.c"))
	t.Run("empty levels", fn([]string{"a", "b", "", "d"}, "a.b.d"))
	t.Run("invalid level", fn([]string{"a", ".", "b", "c"}, "a.b.c"))
	t.Run("leading / trailing", fn([]string{"a", ".b.", "c"}, "a.b.c"))
	t.Run("non-alpha", fn([]string{"a", "/", "#", "?"}, "a./.#.?"))
}

func TestLoad(t *testing.T) {
	fn := func(directory string, expected bool) func(t *testing.T) {
		return func(t *testing.T) {
			_, err := NewTranslations(directory, "en", nil).Load()
			got := (err == nil)

			if got != expected {
				t.Fatalf("expected %v, got %v: %v", expected, got, err)
			}
		}
	}

	t.Run("no file", fn(Validity+"noooooo", false))
	t.Run("invalid file format", fn(Validity+"invalid_file_format", false))
	t.Run("invalid json", fn(Validity+"invalid_json", false))
	t.Run("invalid key #1", fn(Validity+"invalid_key_1", false))
	t.Run("invalid key #2", fn(Validity+"invalid_key_2", false))
	t.Run("duplicate key", fn(Validity+"duplicate_key", false))
	t.Run("invalid lang", fn(Validity+"invalid_lang", false))
	t.Run("invalid intermediate #1", fn(Validity+"invalid_intermediate_1", false))
	t.Run("invalid intermediate #2", fn(Validity+"invalid_intermediate_2", false))
	t.Run("invalid intermediate #3", fn(Validity+"invalid_intermediate_3", false))
	t.Run("invalid translation type #1", fn(Validity+"invalid_translation_type_1", false))
	t.Run("invalid translation type #2", fn(Validity+"invalid_translation_type_2", false))
	t.Run("empty", fn(Validity+"empty", false))
	t.Run("empty translation", fn(Validity+"empty_translation", false))
	t.Run("valid", fn(Validity+"valid", true))
}

func TestNumberTranslations(t *testing.T) {
	fn := func(directory string, expected int) func(t *testing.T) {
		return func(t *testing.T) {
			translations, err := NewTranslations(directory, "en", nil).Load()
			if err != nil {
				t.Fatal(err)
			}

			store := translations.translations[translations.defaultLanguage]
			if len(store) != expected {
				t.Fatalf("expected %v translations, got %v", expected, len(store))
			}
		}
	}

	t.Run("first", fn(Count+"first", 9))
	t.Run("second", fn(Count+"second", 5))
}

func TestNumberIntermediates(t *testing.T) {
	fn := func(directory string, key string, expected int) func(t *testing.T) {
		return func(t *testing.T) {
			translations, err := NewTranslations(directory, "en", nil).Load()
			if err != nil {
				t.Fatal(err)
			}

			store := translations.translations[translations.defaultLanguage]
			if _, ok := store[Key(key)]; !ok {
				t.Fatalf("could not find key %q", key)
			}

			translation := store[Key(key)]
			if len(translation.Intermediates) != expected {
				t.Fatalf("expected %v translations for %q, got %v", expected, key, len(translation.Intermediates))
			}
		}
	}

	t.Run("first #1", fn(Count+"first", "whoami", 1))
	t.Run("first #2", fn(Count+"first", "expired", 2))
	t.Run("first #3", fn(Count+"first", "tyson.defeated", 1))

	t.Run("second #1", fn(Count+"second", "hi", 3))
	t.Run("second #2", fn(Count+"second", "whoami", 0))
	t.Run("second #3", fn(Count+"second", "expired", 0))
	t.Run("second #4", fn(Count+"second", "tyson.defeated", 0))
}
