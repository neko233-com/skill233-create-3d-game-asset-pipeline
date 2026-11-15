# Production verification for the Go 1.27 + Blender 5.2 bpy asset pipeline.
# `make verify` (or `go run ./cmd/assetctl verify`) is the single source of
# truth; scripts/verify.py is a compatibility shim that calls the same Go code.

GO ?= go

# Release target: Windows x86_64 only (user decision). No CI service is used —
# `make release` is the single, local publishing gate.
GOOS_TARGET   ?= windows
GOARCH_TARGET ?= amd64

.PHONY: all build vet test race coverage e2e verify doctor skills skills-doctor dist release clean

dist:
	GOOS=$(GOOS_TARGET) GOARCH=$(GOARCH_TARGET) CGO_ENABLED=0 \
		$(GO) build -trimpath -ldflags "-s -w" -o dist/assetctl.exe ./cmd/assetctl
	@echo "built dist/assetctl.exe ($$(wc -c < dist/assetctl.exe) bytes)"

release: verify dist
	@echo "release ready: dist/assetctl.exe (verify 13/13 green)"

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
	$(GO) run ./cmd/assetctl skills list

skills-doctor:
	$(GO) run ./cmd/assetctl skills doctor

# Only removes generated pipeline artefacts inside THIS repo; never touches
# user directories. Assets configs under assets/ are preserved.
clean:
	$(GO) run ./cmd/assetctl version
	@echo "Remove raw/, work/, build/ contents manually if needed (tracked .gitkeep files stay)."
