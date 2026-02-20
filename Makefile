.PHONY: build run clean

VERSION ?= 0.1.0
BINARY = bin/lctl
LDFLAGS = -ldflags "-X github.com/kkz6/launchctl/cmd.Version=$(VERSION)"

build:
	go build $(LDFLAGS) -o $(BINARY) .

run: build
	./$(BINARY)

clean:
	rm -rf bin/
