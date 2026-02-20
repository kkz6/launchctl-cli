.PHONY: build run clean install uninstall

VERSION ?= 0.1.0
BINARY = bin/lctl
LDFLAGS = -ldflags "-X github.com/kkz6/launchctl/cmd.Version=$(VERSION)"
PREFIX ?= /usr/local

build:
	go build $(LDFLAGS) -o $(BINARY) .

run: build
	./$(BINARY)

install: build
	install -d $(PREFIX)/bin
	install -m 755 $(BINARY) $(PREFIX)/bin/lctl

uninstall:
	rm -f $(PREFIX)/bin/lctl

clean:
	rm -rf bin/
