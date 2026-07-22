.PHONY: build run test clean install install-man uninstall

PREFIX ?= /usr/local
MAN_DIR ?= $(PREFIX)/share/man/man1
BIN = difi

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build:
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BIN) ./cmd/difi

run: build
	./$(BIN)

test:
	go test ./...

clean:
	rm -f $(BIN)

install: build install-man
	@cp $(BIN) $(PREFIX)/bin/$(BIN)
	@echo "difi installed to $(PREFIX)/bin/$(BIN)"
	@echo "Run 'man difi' to view the manual."

install-man:
	@mkdir -p $(MAN_DIR)
	@cp man/man1/difi.1 $(MAN_DIR)/difi.1
	@echo "Man page installed to $(MAN_DIR)/difi.1"

uninstall:
	@rm -f $(PREFIX)/bin/$(BIN)
	@rm -f $(MAN_DIR)/difi.1
	@echo "difi uninstalled."
