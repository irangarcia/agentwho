PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
VERSION ?= dev
LDFLAGS ?= -X github.com/irangarcia/agentwho/internal/cli.Version=$(VERSION)

.PHONY: build test vet lint install uninstall

build:
	mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/agentwho ./cmd/agentwho

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run

install: build
	mkdir -p "$(BINDIR)"
	install -m 0755 bin/agentwho "$(BINDIR)/agentwho"
	"$(BINDIR)/agentwho" install --modify-shell

uninstall:
	agentwho uninstall
