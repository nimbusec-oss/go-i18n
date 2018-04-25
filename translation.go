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
	// Prefix marks the begin of a placeholder being used for i18n interpolation
	Prefix = "{{"
	// Suffix marks the end of a placeholder being used for i18n interpolation
	Suffix = "}}"
)

// Language is the code abbreviation of language
type Language string

// Valid verifies the validity of a language allowing only two letter codes
func (lang Language) Valid() bool {
	return len(lang) == 2 &&
		unicode.IsLetter(rune(lang[0])) &&
		unicode.IsLetter(rune(lang[1]))
}

type Key string

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

type Intermediate string

func (i Intermediate) Format() string {
	return Prefix + string(i) + Suffix
}

type Translation struct {
	Message       string
	Intermediates []Intermediate
}

// Store is a map where a key maps to a translation
type Store map[Key]Translation

// Translations are a collection of language translations represented by key value structure
// Upon translating it will attempt to retrieve the target language from a given source function,
// rolling back to the default language on failure. The translations are loaded during intialization
// from a defined directory
type Translations struct {
	directory       string
	defaultLanguage Language
	languageFn      func() string
	translations    map[Language]Store
}

// NewTranslations initializes a new translations object
func NewTranslations(directory string, defaultLanguage string, languageFn func() string) Translations {
	return Translations{
		directory:       directory,
		defaultLanguage: Language(defaultLanguage),
		languageFn:      languageFn,
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

		// allow only 2-letter language code file name
		lang := Language(strings.ToLower(strings.TrimSuffix(file, extension)))
		if !lang.Valid() {
			return fmt.Errorf("invalid file naming scheme %q, allowed are only two letter codes", lang)
		}

		b, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%v for %q", err, lang)
		}

		var deserialized map[string]interface{}
		err = json.Unmarshal(b, &deserialized)
		if err != nil {
			return fmt.Errorf("%v for %q", err, lang)
		}

		store := make(Store)

		// flatten the nested json objects & combining the key fragments into a complete key string
		var flatten func(Key, map[string]interface{}) error

		flatten = func(rootKey Key, data map[string]interface{}) error {
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
					intermediates, err := parseIntermediates(message)
					if err != nil {
						return fmt.Errorf("%v with key %q", err, rootKey)
					}

					store[rootKey] = Translation{
						Message:       message,
						Intermediates: intermediates,
					}

				case map[string]interface{}:
					err := flatten(rootKey, value.(map[string]interface{}))
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

		trl.translations[lang] = store
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

// parseIntermediates extracts the intermediates in the given translation message
// It allows arbitrary names, prohibiting only empty names.
func parseIntermediates(message string) ([]Intermediate, error) {
	var intermediates []Intermediate

	if strings.Count(message, Prefix) != strings.Count(message, Suffix) {
		return []Intermediate{}, errors.New("invalid format of intermediates")
	}

	parts := strings.Split(message, Prefix)[1:]
	for _, part := range parts {
		i := strings.Index(part, Suffix)
		if i == -1 {
			return []Intermediate{}, errors.New("invalid format of intermediates, must end with " + Suffix)
		}

		intermediate := Intermediate(strings.TrimSpace(part[:i]))
		if intermediate == "" {
			return []Intermediate{}, errors.New("empty intermediate")
		}

		intermediates = append(intermediates, intermediate)
	}
	return intermediates, nil
}

// createIntermediateLookup attempts to resolve a list non-typed parameters
// into a lookup structure putting each odd indexed parameter as key (assuming it to be string)
// and each even indexed non-typed parameter as value
func createIntermediateLookup(parameter []interface{}) (map[Intermediate]interface{}, error) {
	if len(parameter)%2 != 0 {
		return nil, errors.New("invalid dict call")
	}
	dict := make(map[Intermediate]interface{}, len(parameter)/2)
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

// Translate returns the appropriate translation for a given key, interpolating
// the passed parameter values assuming the intermediates
// match the parameter keys injectively.
func (trl Translations) Translate(k string, params ...interface{}) (template.HTML, error) {
	key := Key(k)

	lookup, err := createIntermediateLookup(params)
	if err != nil {
		return "", err
	}

	// try to get valid language from sourceFn; otherwise, roll back to default
	lang := trl.defaultLanguage

	if trl.languageFn != nil {
		if targetLang := Language(trl.languageFn()); targetLang.Valid() {
			lang = targetLang
		}
	}

	if _, ok := trl.translations[lang]; !ok {
		return "", fmt.Errorf("unknown language %q", lang)
	}
	if _, ok := trl.translations[lang][key]; !ok {
		return "", fmt.Errorf("unknown key %q", key)
	}
	translation := trl.translations[lang][key]
	message := translation.Message

	// replace intermediates with passed params
	for _, intermediate := range translation.Intermediates {
		if _, ok := lookup[intermediate]; !ok {
			return "", fmt.Errorf("parameter required for intermediate in translation %q: %q", key, intermediate)
		}

		// escape content of intermediates
		value := html.EscapeString(fmt.Sprintf("%v", lookup[intermediate]))
		message = strings.Replace(message, intermediate.Format(), value, -1)
	}

	// interpret message string as plain HTML allowing tags
	return template.HTML(message), nil
}
