## Go-I18N

[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/nimbusec-oss/go-i18n/blob/master/LICENSE)
[![Go Doc](https://godoc.org/github.com/nimbusec-oss/go-i18n?status.svg)](https://godoc.org/github.com/nimbusec-oss/go-i18n)

## Overview
Go-i18n is a internationalization library for golang using the i18next json format. 

## Features
* named intermediates

## Installation
To install this package, run:
```
go get github.com/nimbusec-oss/go-i18n
```

## Documentation

**Load translations** 
```
t, err := i18n.NewTranslations("<dir>", "en", nil).Load()
```

**Add to FuncMap**
```
template.FuncMap{"T":t.Translate,}
```

**Use in template**
```
{{ T "<translationKey>" }}
```
