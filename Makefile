# Production verification for the Go 1.27 + Blender 5.2 bpy asset pipeline.
# `make verify` (or `go run ./cmd/assetctl verify`) is the single source of
# truth; scripts/verify.py is a compatibility shim that calls the same Go code.

GO ?= go

.PHONY: all build vet test race coverage e2e verify doctor skills clean

all: verify

build:
	$(GO) build ./...

vet:
	$(GO) vet ./...

test:
	$(GO) test ./... -count=1

race:
	$(GO) test -race ./... -count=1

coverage:
	$(GO) test ./... -coverprofile=coverage.out -count=1

e2e:
	$(GO) test ./internal/runner -run TestE2E -v -count=1

verify:
	$(GO) run ./cmd/assetctl verify

doctor:
	$(GO) run ./cmd/assetctl doctor

skills:
	@echo "参考 skills 文档见 skills/README.md；执行入口由 assetctl skills 提供。"

# Only removes generated pipeline artefacts inside THIS repo; never touches
# user directories. Assets configs under assets/ are preserved.
clean:
	$(GO) run ./cmd/assetctl version
	@echo "Remove raw/, work/, build/ contents manually if needed (tracked .gitkeep files stay)."
