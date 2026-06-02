.PHONY: build test vet lint install-local clean

BINARY := zen-sync
PKG    := github.com/gustavoguarda/zen-sync

build:
	go build -o $(BINARY) ./cmd/zen-sync

test:
	go test -race -cover ./...

vet:
	go vet ./...

lint:
	golangci-lint run

install-local: build
	mv $(BINARY) $(HOME)/.local/bin/$(BINARY)

clean:
	rm -f $(BINARY)
	rm -rf dist/
