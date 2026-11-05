VERSION ?= 0.1.0
LDFLAGS := -s -w -X github.com/madebyjonny/forge/internal/scaffold.Version=$(VERSION)
PREFIX  ?= /usr/local

.PHONY: build install install-go test clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/forge .

# Copy the binary onto your PATH (may need sudo for /usr/local).
install: build
	install -m 0755 bin/forge $(PREFIX)/bin/forge

# Or install into $GOBIN / ~/go/bin.
install-go:
	go install -ldflags "$(LDFLAGS)" .

test:
	go test ./...

clean:
	rm -rf bin
