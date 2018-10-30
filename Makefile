
.DEFAULT_GOAL := test

.PHONY: test
test:
	go test -v -race -cover ./...

.PHONY: bench
bench:
	go test -v -run - -bench . -benchmem ./...

.PHONY: lint
lint:
	# Ignore grep's exit code since no match returns 1.
	-if [[ ! $TRAVIS_GO_VERSION = 1.8* ]]; then echo 'linting...' ; golint ./... ; fi

.PHONY: vet
vet:
	go vet ./...

.PHONY: all
all: vet lint test bench

.PHONY: example
