.DEFAULT_GOAL := help

.PHONY: help build install uninstall test test-coverage lint fmt vet tidy install-tools clean audit portability-check run all

help:
	@printf '%s\n' \
		'Available targets:' \
		'  build             Build the binary' \
		'  install           Install the binary to GOBIN/GOPATH/bin' \
		'  uninstall         Remove the installed binary' \
		'  test              Run the test suite' \
		'  test-coverage     Run tests and open the coverage report' \
		'  lint              Run golangci-lint' \
		'  fmt               Format code with gofumpt and goimports' \
		'  vet               Run go vet' \
		'  tidy              Tidy and verify modules' \
		'  install-tools     Install gofumpt, goimports, and golangci-lint' \
		'  clean             Remove build artifacts and coverage output' \
		'  audit             Run all quality checks' \
		'  portability-check  Check portability docs freshness' \
		'  run               Build and run the binary' \
		'  all               Run audit and build'

build:
	@just build

install:
	@just install

uninstall:
	@just uninstall

test:
	@just test

test-coverage:
	@just test-coverage

lint:
	@just lint

fmt:
	@just fmt

vet:
	@just vet

tidy:
	@just tidy

install-tools:
	@just install-tools

clean:
	@just clean

audit:
	@just audit

portability-check:
	@just portability-check

run:
	@just run

all:
	@just all
