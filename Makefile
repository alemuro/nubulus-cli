BINARY_NAME=nubulus
VERSION=1.0.0
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X github.com/alemuro/nubulus-cli/cmd.Version=$(VERSION) -X github.com/alemuro/nubulus-cli/cmd.Commit=$(COMMIT) -X github.com/alemuro/nubulus-cli/cmd.Date=$(DATE)"

.PHONY: all build test clean install

all: test build

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

test:
	go test -v ./...

clean:
	rm -f $(BINARY_NAME)

install: build
	install -d $(HOME)/.local/bin
	install -m 755 $(BINARY_NAME) $(HOME)/.local/bin/$(BINARY_NAME)
