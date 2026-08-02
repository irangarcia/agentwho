PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin

.PHONY: build test vet install uninstall

build:
	go build -o bin/agentwho ./cmd/agentwho

test:
	go test ./...

vet:
	go vet ./...

install: build
	mkdir -p "$(BINDIR)"
	install -m 0755 bin/agentwho "$(BINDIR)/agentwho"
	"$(BINDIR)/agentwho" install --modify-shell

uninstall:
	agentwho uninstall
