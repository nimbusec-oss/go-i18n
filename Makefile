# --- test
.PHONY: test
test:
	mkdir -p _goTestOutput
	docker run --rm \
		-u $(shell id -u) \
		-v ${PWD}:/go/src/github.com/nimbusec-oss/go-i18n \
		golang:1.9 /bin/bash -c "\
		go test -v github.com/nimbusec-oss/go-i18n/..." > _goTestOutput/test.log
