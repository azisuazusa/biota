TAILWIND_VERSION ?= v4.3.5
TAILWIND_BIN := ./bin/tailwindcss

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)
ifeq ($(UNAME_S),Darwin)
  TW_OS := macos
else
  TW_OS := linux
endif
ifneq (,$(filter $(UNAME_M),arm64 aarch64))
  TW_ARCH := arm64
else
  TW_ARCH := x64
endif

.PHONY: generate test run-lint run mocks docker-build tailwind-install

generate:
	templ generate && $(TAILWIND_BIN) -i static/css/input.css -o static/css/app.css --minify

test:
	go test -count=1 -failfast ./...

run-lint:
	golangci-lint run ./...

run:
	make generate && DATA_DIR=./.data go run ./app/http

mocks:
	mockery

docker-build:
	docker build -t biota-insight .

tailwind-install:
	mkdir -p bin
	curl -sSfL -o $(TAILWIND_BIN) https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-$(TW_OS)-$(TW_ARCH)
	chmod +x $(TAILWIND_BIN)
