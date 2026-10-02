package i18n

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const (
	// IntermediatePrefix marks the begin of a placeholder being used for i18n interpolation
	IntermediatePrefix = "{{"
	// IntermediateSuffix marks the end of a placeholder being used for i18n interpolation
	IntermediateSuffix = "}}"
	// NestedPrefix marks the opening of nested translation
	NestedPrefix = "$t("
	// NestedSuffix marks the end of nested translation
	NestedSuffix = ")"
)

// TranslationFunc is a type alias for a translation function
// It takes a translation key and optional parameters and returns the appropriate translation
// An error is returned if the key cannot be translated
type TranslationFunc func(k string, params ...any) (template.HTML, error)

// Translations are a collection of language translations represented by key value structure
// Upon translating it will attempt to retrieve the target language from a given source function,
// rolling back to the default language on failure. The translations are loaded during intialization
// from a defined directory
type Translations struct {
	directory       string
	defaultLanguage Language
	translations    map[Language]Store
}

// Language is the code abbreviation of language
type Language string

// Valid verifies the validity of a language allowing only two letter codes
func (lang Language) Valid() bool {
	return len(lang) == 2 &&
		unicode.IsLetter(rune(lang[0])) &&
		unicode.IsLetter(rune(lang[1]))
}

// Store is a map where a key maps to a translation
type Store map[Key]Translation

// Key is a unique identifier for a translation.
// A Key comprises multiple fragments seperated by a "."
type Key string

// Append takes a fragment s and appends
// it to the current key. If the current key
// is empty, s is the root fragment
func (k Key) Append(s string) Key {
	s = strings.Trim(s, ".")
	if s == "" {
		return k
	}

	if k != "" {
		return Key(string(k) + "." + s)
	}
	return Key(s)
}

func (k Key) String() string {
	return string(k)
}

// Translation defines the translated message
// for a given key which contains context-dependent intermediates as placeholder
// Nested translations are a string of keys to be resolved by GenerateTranslate
type Translation struct {
	Message       string
	Intermediates []Intermediate
	Nested        []NestedTranslation
}

// NestedTranslation represents a reference (key) to another translation.
type NestedTranslation struct {
	Key    Key
	Format string
}

// Intermediate is a named placeholder within
// a translation which may be replaced by a
// context-dependent value
type Intermediate string

// Format returns the intermediate in i18next notation
// e.g {{hello}}

func (i Intermediate) Format() string {
	return IntermediatePrefix + string(i) + IntermediateSuffix
}

// NewTranslations initializes a new translations object
func NewTranslations(directory string, defaultLanguage string) Translations {
	return Translations{
		directory:       directory,
		defaultLanguage: Language(defaultLanguage),
	}
}

// Load processes all language files of the defined directory and parses it into
// a kv structure keyed by the language code. It fetches all files in the directory
// using their base name as language identifier. The files are expected to be of JSON format.
// Load allows nested translations in the file meaning the key must not be denoted
// in a single form but can be splitted along the nesting levels (it follows the i18next standard).
// It will recursively summarize these keys into a full one, saving each value under the appropriate
// full key and return a flattened structure.
func (trl Translations) Load() (Translations, error) {
	if !trl.defaultLanguage.Valid() {
		return Translations{}, errors.New("invalid default language, must follow two letter code")
	}

	trl.translations = make(map[Language]Store)

	err := filepath.Walk(trl.directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		extension := filepath.Ext(path)
		if extension != ".json" {
			return nil
		}

		_, file := filepath.Split(path)

		_, language, _ := cutLast(strings.TrimSuffix(file, extension), "-")
		lang := Language(strings.ToLower(language))
		// allow only 2-letter language code file name
		if !lang.Valid() {
			return fmt.Errorf("invalid file naming scheme %q, allowed are only two letter codes", lang)
		}

		b, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%v for %q", err, lang)
		}

		var deserialized map[string]any
		err = json.Unmarshal(b, &deserialized)
		if err != nil {
			return fmt.Errorf("%v for %q", err, lang)
		}

		store := make(Store)

		// flatten the nested json objects & combining the key fragments into a complete key string
		var flatten func(Key, map[string]any) error

		flatten = func(rootKey Key, data map[string]any) error {
			if len(data) == 0 {
				return fmt.Errorf("invalid translation for %q", rootKey)
			}

			for key, value := range data {
				if key == "" {
					return errors.New("invalid key, should not be empty")
				}

				// append key fragment to root key
				rootKey := rootKey.Append(key)

				switch t := value.(type) {
				case string:
					message := value.(string)

					// parse the intermediates (if existing) of message string
					// for fail-safety
					// create a list of nested translations for recursive lookup
					intermediates, nested, err := parseIntermediates(message)
					if err != nil {
						return fmt.Errorf("%v with key %q", err, rootKey)
					}

					store[rootKey] = Translation{
						Message:       message,
						Intermediates: intermediates,
						Nested:        nested,
					}

				case map[string]any:
					err := flatten(rootKey, value.(map[string]any))
					if err != nil {
						return err
					}

				default:
					return fmt.Errorf("invalid type %T in translation file, only string or objects as values allowed", t)
				}
			}

			return nil
		}
		var k Key
		err = flatten(k, deserialized)
		if err != nil {
			return fmt.Errorf("%v for %q", err, lang)
		}

		// within the translations file, there must be at least one translation
		if len(store) == 0 {
			return fmt.Errorf("no translations found for %q", lang)
		}

		if _, ok := trl.translations[lang]; ok {
			for key, value := range store {
				trl.translations[lang][key] = value
			}
		} else {
			trl.translations[lang] = store
		}

		return nil
	})
	if err != nil {
		return Translations{}, err
	}

	if _, ok := trl.translations[trl.defaultLanguage]; !ok {
		return Translations{}, fmt.Errorf("no translations found for default language")
	}

	return trl, nil
}

// parseIntermediates extracts intermediates and nested translations from the translation
// Nested translation references are only parsed here.
// They are resolved recursively by GenerateTranslate

// This logic and flow is copied from github.com i18n.next, the parser and stack
// were recieved help from Charles GPT

func parseIntermediates(message string) ([]Intermediate, []NestedTranslation, error) {
	var intermediates []Intermediate
	var nested []NestedTranslation

	for i := 0; i < len(message); {
		switch {
		case strings.HasPrefix(message[i:], IntermediatePrefix):
			value, next, err := scanFormat(
				message,
				i,
				IntermediatePrefix,
				IntermediateSuffix,
			)
			if err != nil {
				return nil, nil, fmt.Errorf(
					"invalid intermediate: %w",
					err,
				)
			}

			intermediates = append(
				intermediates,
				Intermediate(value),
			)

			i = next

		case strings.HasPrefix(message[i:], NestedPrefix):
			value, next, err := scanFormat(
				message,
				i,
				NestedPrefix,
				NestedSuffix,
			)
			if err != nil {
				return nil, nil, fmt.Errorf(
					"invalid nested translation: %w",
					err,
				)
			}

			rawKey := value

			// We handle quoted and unquoted nested keys
			//   $t(foo.bar)
			//   $t('foo.bar')
			//   $t("foo.bar")

			if len(rawKey) >= 2 {
				// base case no quotes
				first := rawKey[0]
				last := rawKey[len(rawKey)-1]
				// quoted keys
				if (first == '\'' && last == '\'') || (first == '"' && last == '"') {
					rawKey = strings.TrimSpace(rawKey[1 : len(rawKey)-1])
				}
			}

			if rawKey == "" {
				return nil, nil, errors.New(
					"empty nested translation key",
				)
			}

			nested = append(nested, NestedTranslation{
				Key:    Key(rawKey),
				Format: message[i:next],
			})

			i = next

		default:
			i++
		}
	}

	return intermediates, nested, nil
}

func scanFormat(message string, start int, prefix string, suffix string) (value string, next int, err error) {
	contentStart := start + len(prefix)

	offset := strings.Index(message[contentStart:], suffix)
	if offset == -1 {
		return "", start, fmt.Errorf(
			"must end with %q",
			suffix,
		)
	}

	end := contentStart + offset

	value = strings.TrimSpace(message[contentStart:end])
	if value == "" {
		return "", start, errors.New("empty value")
	}

	return value, end + len(suffix), nil
}

// createIntermediateLookup attempts to resolve a list non-typed parameters
// into a lookup structure putting each odd indexed parameter as key (assuming it to be string)
// and each even indexed non-typed parameter as value
func createIntermediateLookup(parameter []any) (map[Intermediate]any, error) {
	if len(parameter)%2 != 0 {
		return nil, errors.New("invalid dict call")
	}
	dict := make(map[Intermediate]any, len(parameter)/2)
	for i := 0; i < len(parameter); i += 2 {
		_, ok := parameter[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		key := Intermediate(parameter[i].(string))
		dict[key] = parameter[i+1]
	}
	return dict, nil
}

// GenerateDefaultTranslate returns a translate function for the default language.
func (trl Translations) GenerateDefaultTranslate() TranslationFunc {
	return trl.GenerateTranslate(string(trl.defaultLanguage))
}

// GenerateTranslate returns a translate function for a specific language that translates a given key, interpolating
// the passed parameter values assuming the intermediates
// match the parameter keys injectively.
func (trl Translations) GenerateTranslate(targetLang string) TranslationFunc {
	lang := Language(targetLang)
	if !lang.Valid() {
		lang = trl.defaultLanguage
	}

	return func(k string, params ...any) (template.HTML, error) {
		lookup, err := createIntermediateLookup(params)
		if err != nil {
			return "", err
		}
		// Using a stack to prevent circular referneces as per i18n.next
		//   label.one -> $t('label.two')
		//   label.two -> $t('label.one')

		stack := make(map[Key]bool)

		var translate func(key Key) (template.HTML, error)

		translate = func(key Key) (template.HTML, error) {
			if stack[key] {
				return "", fmt.Errorf("circular translation reference %q", key)
			}

			if _, ok := trl.translations[lang]; !ok {
				return "", fmt.Errorf("unknown language %q", lang)
			}

			translation, ok := trl.translations[lang][key]
			if !ok {
				return "", fmt.Errorf("unknown key %q", key)
			}

			stack[key] = true
			defer delete(stack, key)

			message := translation.Message

			// Replace intermediates with passed params.
			for _, intermediate := range translation.Intermediates {
				value, ok := lookup[intermediate]
				if !ok {
					return "", fmt.Errorf(
						"parameter required for intermediate in translation %q: %q",
						key,
						intermediate,
					)
				}

				// Escape content of intermediates.
				escapedValue := html.EscapeString(fmt.Sprintf("%v", value))
				message = strings.Replace(
					message,
					intermediate.Format(),
					escapedValue,
					1,
				)
			}

			// Resolve nested translations.
			//
			// Nested translations are resolved after interpolation,
			// as per github.com i18n.next/src/interpolator.js
			for _, nested := range translation.Nested {
				value, err := translate(nested.Key)
				if err != nil {
					return "", err
				}

				message = strings.Replace(
					message,
					nested.Format,
					string(value),
					1,
				)
			}

			// Interpret message string as plain HTML allowing tags.
			return template.HTML(message), nil
		}

		return translate(Key(k))
	}
}

// AvailableLanguages returns a list of available languages
// that were discovered in the language file directory.
func (trl Translations) AvailableLanguages() []string {
	availableLanguages := []string{}
	for lang := range trl.translations {
		availableLanguages = append(availableLanguages, string(lang))
	}

	return availableLanguages
}

func cutLast(s, sep string) (before, after string, found bool) {
	if i := strings.LastIndex(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return "", s, false
}

func T(fn TranslationFunc, key string, intermediates ...any) (string, error) {
	translated, err := fn(key, intermediates...)
	if err != nil {
		return "??" + key + "??", err
	}
	return string(translated), nil
}
