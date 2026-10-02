package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestTranslations_LifeCycle(t *testing.T) {
	t.Parallel()

	enJSON := `{
		"label": {
			"welcome": "Welcome",
			"hello": "Hello, {{name}}!",
			"product": "application",
			"description": "This is our $t(label.product) $t(label.product).",
			"greeting": "$t('label.welcome'), {{name}}!",
			"footer": "Thank you for using our $t(label.product).",
			"account": "{{name}}'s account",
			"save": "Save",
			"delete": "Delete",
			"confirmation": "{{name}}, are you sure you want to $t(label.delete) this item?"
		}
	}`

	deJSON := `{
		"label": {
			"welcome": "Willkommen",
			"hello": "Hallo, {{name}}!",
			"product": "Anwendung",
			"description": "Dies ist unsere $t(label.product).",
			"greeting": "$t('label.welcome'), {{name}}!",
			"footer": "Danke, dass Sie unsere $t(label.product) verwenden.",
			"account": "{{name}}s Konto",
			"save": "Speichern",
			"delete": "Löschen",
			"confirmation": "{{name}}, möchten Sie diesen Eintrag wirklich $t(label.delete)?"
		}
	}`

	const expectedLanguages = 2
	const expectedKeysPerLanguage = 10

	t.Logf(
		"STEP 1: create translation fixture | expected languages=%d keys_per_language=%d",
		expectedLanguages,
		expectedKeysPerLanguage,
	)

	dir := t.TempDir()

	files := map[string]string{
		"en.json": enJSON,
		"de.json": deJSON,
	}

	for filename, content := range files {
		path := filepath.Join(dir, filename)

		t.Logf("STEP 1.%s: write JSON | expected path=%q", filename, path)

		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("FAIL: STEP 1.%s: write JSON | expected success | actual error=%v", filename, err)
		}

		t.Logf("PASS: STEP 1.%s: write JSON | actual path=%q", filename, path)
	}

	t.Logf("STEP 2: create Translations object | expected directory=%q default_language=%q", dir, "en")

	translations := NewTranslations(dir, "en")

	t.Logf("STEP 2: create Translations object | actual directory=%q default_language=%q translations=%#v", translations.directory, translations.defaultLanguage, translations.translations)

	t.Logf("STEP 3: load translations | expected languages=[en de]")

	t.Logf("STEP 3: before Load | translations=%#v", translations.translations)

	var err error

	translations, err = translations.Load()
	if err != nil {
		t.Fatalf("FAIL: STEP 3: Load | expected success | actual error=%v", err)
	}

	t.Logf("STEP 3: after Load | translations=%#v", translations.translations)

	availableLanguages := translations.AvailableLanguages()

	t.Logf("STEP 3: loaded languages | expected count=%d | actual count=%d | actual=%v", expectedLanguages, len(availableLanguages), availableLanguages)

	if len(availableLanguages) != expectedLanguages {
		t.Errorf("FAIL: STEP 3: loaded languages | expected count=%d | actual count=%d", expectedLanguages, len(availableLanguages))
	} else {
		t.Logf("PASS: STEP 3: loaded languages | expected count=%d | actual count=%d", expectedLanguages, len(availableLanguages))
	}

	t.Logf("STEP 4: validate language stores | expected languages=[en de], keys_per_language=%d", expectedKeysPerLanguage)

	for _, language := range []Language{"en", "de"} {
		store, ok := translations.translations[language]

		t.Logf("STEP 4: language=%q | expected exists=true | actual exists=%v", language, ok)

		if !ok {
			t.Errorf("FAIL: STEP 4: language=%q | expected exists=true | actual exists=false", language)
			continue
		}

		t.Logf("STEP 4: language=%q | expected keys=%d | actual keys=%d", language, expectedKeysPerLanguage, len(store))

		if len(store) != expectedKeysPerLanguage {
			t.Errorf("FAIL: STEP 4: language=%q | expected keys=%d | actual keys=%d", language, expectedKeysPerLanguage, len(store))
			continue
		}

		t.Logf("PASS: STEP 4: language=%q | keys=%d", language, len(store))
	}

	t.Logf("STEP 5: inspect loaded English translations")

	enStore, ok := translations.translations[Language("en")]
	if !ok {
		t.Fatalf(
			"FAIL: STEP 5: English store | expected store to exist",
		)
	}

	expectedKeys := []Key{
		"label.welcome",
		"label.hello",
		"label.product",
		"label.description",
		"label.greeting",
		"label.footer",
		"label.account",
		"label.save",
		"label.delete",
		"label.confirmation",
	}

	for _, key := range expectedKeys {
		translation, ok := enStore[key]

		t.Logf("STEP 5: key=%q | expected exists=true | actual exists=%v", key, ok)

		if !ok {
			t.Errorf("FAIL: STEP 5: key=%q | expected exists=true | actual exists=false", key)
			continue
		}

		t.Logf("STEP 5: key=%q | message=%q | intermediates=%#v", key, translation.Message, translation.Intermediates)

		t.Logf("PASS: STEP 5: key=%q", key)
	}

	t.Logf("STEP 6: generate English translator | expected language=%q", "en")

	translateEN := translations.GenerateTranslate("en")

	if translateEN == nil {
		t.Fatalf("FAIL: STEP 6: English translator | expected non-nil translator")
	}

	t.Logf("PASS: STEP 6: English translator created")

	t.Logf("STEP 7: generate German translator | expected language=%q", "de")

	translateDE := translations.GenerateTranslate("de")

	if translateDE == nil {
		t.Fatalf("FAIL: STEP 7: German translator | expected non-nil translator")
	}

	t.Logf("PASS: STEP 7: German translator created")

	tests := []struct {
		name      string
		language  string
		translate TranslationFunc
		key       string
		params    []any
		expected  string
	}{
		{
			name:      "EN basic translation",
			language:  "en",
			translate: translateEN,
			key:       "label.welcome",
			expected:  "Welcome",
		},
		{
			name:      "EN with interpolated translation",
			language:  "en",
			translate: translateEN,
			key:       "label.hello",
			params:    []any{"name", "John"},
			expected:  "Hello, John!",
		},
		{
			name:      "EN with nested translation / repeated nesting in single depth key",
			language:  "en",
			translate: translateEN,
			key:       "label.description",
			expected:  "This is our application application.",
		},
		{
			name:      "EN with nested and interpolated translation",
			language:  "en",
			translate: translateEN,
			key:       "label.greeting",
			params:    []any{"name", "John"},
			expected:  "Welcome, John!",
		},
		{
			name:      "EN nested translation",
			language:  "en",
			translate: translateEN,
			key:       "label.footer",
			expected:  "Thank you for using our application.",
		},
		{
			name:      "EN nested translation with interpolation",
			language:  "en",
			translate: translateEN,
			key:       "label.confirmation",
			params:    []any{"name", "John"},
			expected:  "John, are you sure you want to Delete this item?",
		},
		{
			name:      "DE plain translation",
			language:  "de",
			translate: translateDE,
			key:       "label.welcome",
			expected:  "Willkommen",
		},
		{
			name:      "DE with interpolated translation",
			language:  "de",
			translate: translateDE,
			key:       "label.hello",
			params:    []any{"name", "Hans"},
			expected:  "Hallo, Hans!",
		},
		{
			name:      "DE with nested translation",
			language:  "de",
			translate: translateDE,
			key:       "label.description",
			expected:  "Dies ist unsere Anwendung.",
		},
		{
			name:      "DE with nested and interpolated translation",
			language:  "de",
			translate: translateDE,
			key:       "label.greeting",
			params:    []any{"name", "Hans"},
			expected:  "Willkommen, Hans!",
		},
	}

	t.Logf(
		"STEP 8: execute translation tests | expected cases=%d",
		len(tests),
	)

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := fmt.Sprintf("STEP 8.%d", i+1)
			t.Logf("%s: translate | language=%q key=%q params=%#v", step, tt.language, tt.key, tt.params)
			t.Logf("%s: expected=%q", step, tt.expected)
			got, err := tt.translate(tt.key, tt.params...)

			if err != nil {
				t.Errorf("FAIL: %s: translate | expected=%q | actual=error: %v", step, tt.expected, err)
				return
			}

			actual := string(got)

			t.Logf("%s: actual=%q", step, actual)

			if actual != tt.expected {
				t.Errorf("FAIL: %s: translate | expected=%q | actual=%q", step, tt.expected, actual)
				return
			}

			t.Logf("PASS: %s: translate | expected=%q | actual=%q", step, tt.expected, actual)
		})
	}

	t.Logf(
		"PASSED - ALL | LOAD | INTERPOLATION | NESTING",
	)
}

func TestTranslations_CircularNestedTranslation(t *testing.T) {
	t.Parallel()

	enJSON := `{
		"label": {
			"first": "First -> $t(label.second)",
			"second": "Second -> $t(label.first)"
		}
	}`

	t.Logf(
		"STEP 1: create fixture | expected first -> second -> first",
	)

	dir := t.TempDir()

	path := filepath.Join(dir, "en.json")

	t.Logf(
		"STEP 1: write JSON | expected path=%q",
		path,
	)

	if err := os.WriteFile(path, []byte(enJSON), 0o644); err != nil {
		t.Fatalf(
			"FAIL: STEP 1: write JSON | expected success | actual error=%v",
			err,
		)
	}

	t.Logf(
		"PASS: STEP 1: write JSON | actual path=%q",
		path,
	)

	t.Logf(
		"STEP 2: create and load translations | expected language=%q",
		"en",
	)

	translations := NewTranslations(dir, "en")

	var err error

	translations, err = translations.Load()
	if err != nil {
		t.Fatalf(
			"FAIL: STEP 2: Load | expected success | actual error=%v",
			err,
		)
	}

	t.Logf(
		"PASS: STEP 2: Load | languages=%v",
		translations.AvailableLanguages(),
	)

	t.Logf(
		"STEP 3: generate English translator | expected language=%q",
		"en",
	)

	translate := translations.GenerateTranslate("en")

	if translate == nil {
		t.Fatalf(
			"FAIL: STEP 3: GenerateTranslate | expected non-nil translator",
		)
	}

	t.Logf(
		"PASS: STEP 3: English translator created",
	)

	t.Logf(
		"STEP 4: translate circular key | expected circular reference error",
	)

	got, err := translate("label.first")

	t.Logf(
		"STEP 4: actual result=%q | actual error=%v",
		string(got),
		err,
	)

	if err == nil {
		t.Fatalf(
			"FAIL: STEP 4: circular translation | expected error | actual error=nil, result=%q",
			string(got),
		)
	}

	expectedError := `circular translation reference "label.first"`

	if err.Error() != expectedError {
		t.Errorf(
			"FAIL: STEP 4: circular translation | expected error=%q | actual error=%q",
			expectedError,
			err.Error(),
		)
		return
	}

	if got != "" {
		t.Errorf(
			"FAIL: STEP 4: circular translation | expected empty result | actual=%q",
			string(got),
		)
		return
	}

	t.Logf(
		"PASS: STEP 4: circular translation | expected error=%q | actual error=%q",
		expectedError,
		err.Error(),
	)

	t.Logf(
		"PASS ALL: circular nested translation is detected without infinite recursion",
	)
}

func TestParseIntermediates(t *testing.T) {
	tests := []struct {
		name              string
		message           string
		wantIntermediates []Intermediate
		wantNested        []NestedTranslation
		wantErr           string
	}{
		{
			name:    "no intermediates",
			message: "plain translation",
		},
		{
			name:              "single intermediate",
			message:           "Hello {{name}}",
			wantIntermediates: []Intermediate{"name"},
		},
		{
			name:              "multiple intermediates",
			message:           "{{greeting}}, {{name}}!",
			wantIntermediates: []Intermediate{"greeting", "name"},
		},
		{
			name:              "intermediate with whitespace",
			message:           "Hello {{ name }}",
			wantIntermediates: []Intermediate{"name"},
		},
		{
			name:    "unquoted nested translation",
			message: "This is $t(foo.bar.baz)",
			wantNested: []NestedTranslation{
				{
					Key:    "foo.bar.baz",
					Format: "$t(foo.bar.baz)",
				},
			},
		},
		{
			name:    "single quoted nested translation",
			message: "This is $t('foo.bar.baz')",
			wantNested: []NestedTranslation{
				{
					Key:    "foo.bar.baz",
					Format: "$t('foo.bar.baz')",
				},
			},
		},
		{
			name:    "double quoted nested translation",
			message: `This is $t("foo.bar.baz")`,
			wantNested: []NestedTranslation{
				{
					Key:    "foo.bar.baz",
					Format: `$t("foo.bar.baz")`,
				},
			},
		},
		{
			name:    "unquoted nested translation with whitespace",
			message: "This is $t( foo.bar.baz )",
			wantNested: []NestedTranslation{
				{
					Key:    "foo.bar.baz",
					Format: "$t( foo.bar.baz )",
				},
			},
		},
		{
			name:    "single quoted nested translation with whitespace",
			message: "This is $t( 'foo.bar.baz' )",
			wantNested: []NestedTranslation{
				{
					Key:    "foo.bar.baz",
					Format: "$t( 'foo.bar.baz' )",
				},
			},
		},
		{
			name:    "double quoted nested translation with whitespace",
			message: `This is $t( "foo.bar.baz" )`,
			wantNested: []NestedTranslation{
				{
					Key:    "foo.bar.baz",
					Format: `$t( "foo.bar.baz" )`,
				},
			},
		},
		{
			name:    "deeply nested translation key",
			message: "$t(label.dsassessmentstep.footer.opening)",
			wantNested: []NestedTranslation{
				{
					Key:    "label.dsassessmentstep.footer.opening",
					Format: "$t(label.dsassessmentstep.footer.opening)",
				},
			},
		},
		{
			name:    "multiple nested translations",
			message: "Hello $t(greeting), $t('name')!",
			wantNested: []NestedTranslation{
				{
					Key:    "greeting",
					Format: "$t(greeting)",
				},
				{
					Key:    "name",
					Format: "$t('name')",
				},
			},
		},
		{
			name:              "intermediates and nested translations",
			message:           "Hello {{name}}, $t(greeting)!",
			wantIntermediates: []Intermediate{"name"},
			wantNested: []NestedTranslation{
				{
					Key:    "greeting",
					Format: "$t(greeting)",
				},
			},
		},
		{
			name:              "multiple intermediates and nested translations",
			message:           "{{greeting}} {{name}} - $t(foo.title) / $t('foo.description')",
			wantIntermediates: []Intermediate{"greeting", "name"},
			wantNested: []NestedTranslation{
				{
					Key:    "foo.title",
					Format: "$t(foo.title)",
				},
				{
					Key:    "foo.description",
					Format: "$t('foo.description')",
				},
			},
		},
		{
			name:    "empty intermediate",
			message: "Hello {{}}",
			wantErr: "invalid intermediate: empty value",
		},
		{
			name:    "unterminated intermediate",
			message: "Hello {{name",
			wantErr: "invalid intermediate: must end with \"}}\"",
		},
		{
			name:    "empty nested translation",
			message: "Hello $t()",
			wantErr: "invalid nested translation: empty value",
		},
		{
			name:    "unterminated nested translation",
			message: "Hello $t(foo.bar",
			wantErr: "invalid nested translation: must end with \")\"",
		},
		{
			name:    "empty quoted nested translation",
			message: "Hello $t('')",
			wantErr: "empty nested translation key",
		},
		{
			name:    "empty double quoted nested translation",
			message: `Hello $t("")`,
			wantErr: "empty nested translation key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr != "" {
				t.Logf("TEST: %s | expected error: %q", tt.name, tt.wantErr)
			} else {
				t.Logf("TEST: %s | expected successful parse", tt.name)
			}

			gotIntermediates, gotNested, err := parseIntermediates(tt.message)

			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("FAIL: %s | expected error %q, got nil", tt.name, tt.wantErr)
					return
				}

				if err.Error() != tt.wantErr {
					t.Errorf(
						"FAIL: %s | expected error %q, got %q", tt.name, tt.wantErr, err.Error())
					return
				}

				t.Logf("PASS: %s | received expected error: %q", tt.name, err.Error())
				return
			}

			if err != nil {
				t.Errorf("FAIL: %s | unexpected error: %v", tt.name, err)
				return
			}

			if !reflect.DeepEqual(gotIntermediates, tt.wantIntermediates) {
				t.Errorf("FAIL: %s | intermediates: expected %#v, got %#v", tt.name, tt.wantIntermediates, gotIntermediates)
				return
			}

			if !reflect.DeepEqual(gotNested, tt.wantNested) {
				t.Errorf("FAIL: %s | nested: expected %#v, got %#v", tt.name, tt.wantNested, gotNested)
				return
			}

			t.Logf("PASS: %s | successful parse: intermediates=%#v nested=%#v", tt.name, gotIntermediates, gotNested)
		})
	}

}
