# --- test
.PHONY: test
test:
	docker run --rm \
		-u $(shell id -u) \
		-v ${PWD}:/go/src/github.com/nimbusec-oss/go-i18n \
		golang:1.9 /bin/bash -c "\
		go test github.com/nimbusec-oss/go-i18n/..."