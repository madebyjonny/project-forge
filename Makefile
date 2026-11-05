VERSION ?= 0.1.0
LDFLAGS := -s -w -X github.com/madebyjonny/forge/internal/scaffold.Version=$(VERSION)
PREFIX  ?= $(HOME)/.local

.PHONY: build install install-go test clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/forge .

# Copy the binary onto your PATH. Defaults to ~/.local/bin (no sudo);
# use `sudo make install PREFIX=/usr/local` for a system-wide install.
install: build
	install -d $(PREFIX)/bin
	install -m 0755 bin/forge $(PREFIX)/bin/forge
	@echo "installed $(PREFIX)/bin/forge"
	@case ":$$PATH:" in *":$(PREFIX)/bin:"*) ;; *) echo "note: add $(PREFIX)/bin to your PATH";; esac

# Or install into $GOBIN / ~/go/bin.
install-go:
	go install -ldflags "$(LDFLAGS)" .

test:
	go test ./...

clean:
	rm -rf bin
