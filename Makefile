BINARY  := grm
PKG     := ./...
VERSION := 0.0.0
MONOVA  := $(shell which monova 2> /dev/null)
LDFLAGS  = -ldflags="-X github.com/jsnjack/grm/cmd.Version=$(VERSION)"

export PATH := $(PATH):$(shell go env GOPATH)/bin

version:
ifdef MONOVA
override VERSION = $(shell monova)
override LDFLAGS = -ldflags="-X github.com/jsnjack/grm/cmd.Version=$(VERSION)"
else
	$(info "Install monova with: grm install jsnjack/monova")
endif

test:
	go test $(PKG)

vet:
	go vet $(PKG)

fmt:
	@command -v goimports >/dev/null 2>&1 || { \
	  echo "goimports is not installed. Install it with:"; \
	  echo "  go install golang.org/x/tools/cmd/goimports@latest"; \
	  exit 1; \
	}
	goimports -w .

lint: vet
	@command -v golangci-lint >/dev/null 2>&1 || { \
	  echo "golangci-lint is not installed. Install it with:"; \
	  echo "  grm install golangci/golangci-lint"; \
	  exit 1; \
	}
	golangci-lint run

check: fmt vet build test lint
	@echo "==> make check: all green"

standards:
	curl -sL https://raw.githubusercontent.com/jsnjack/standards/master/AGENTS.universal.md \
	    -o AGENTS.universal.md
	curl -sL https://raw.githubusercontent.com/jsnjack/standards/master/AGENTS.go.md \
	    -o AGENTS.go.md

bin/$(BINARY): bin/$(BINARY)_linux_amd64
	cp $< $@
	ln -sf bin/$(BINARY) $(BINARY)
bin/$(BINARY)_linux_amd64: version main.go cmd/*.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $@
bin/$(BINARY)_linux_arm64: version main.go cmd/*.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $@
bin/$(BINARY)_darwin_amd64: version main.go cmd/*.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $@
bin/$(BINARY)_darwin_arm64: version main.go cmd/*.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $@

build: bin/$(BINARY) bin/$(BINARY)_linux_amd64 bin/$(BINARY)_linux_arm64 bin/$(BINARY)_darwin_amd64 bin/$(BINARY)_darwin_arm64

# bin/grm is listed first so it lands as the release's first asset — the
# README's `jq -r .assets[0].browser_download_url` install one-liner depends
# on that ordering.
release: build
	tar --transform='s,_.*,,' --transform='s,bin/,,' -cz -f bin/$(BINARY)_linux_amd64.tar.gz bin/$(BINARY)_linux_amd64
	tar --transform='s,_.*,,' --transform='s,bin/,,' -cz -f bin/$(BINARY)_linux_arm64.tar.gz bin/$(BINARY)_linux_arm64
	tar --transform='s,_.*,,' --transform='s,bin/,,' -cz -f bin/$(BINARY)_darwin_amd64.tar.gz bin/$(BINARY)_darwin_amd64
	tar --transform='s,_.*,,' --transform='s,bin/,,' -cz -f bin/$(BINARY)_darwin_arm64.tar.gz bin/$(BINARY)_darwin_arm64
	grm release jsnjack/$(BINARY) \
		-f bin/$(BINARY) \
		-f bin/$(BINARY)_linux_amd64.tar.gz \
		-f bin/$(BINARY)_linux_arm64.tar.gz \
		-f bin/$(BINARY)_darwin_amd64.tar.gz \
		-f bin/$(BINARY)_darwin_arm64.tar.gz \
		-t "v`monova`"

clean:
	rm -rf bin/ $(BINARY)

.PHONY: version release build test vet fmt lint check standards clean
