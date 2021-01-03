PWD:=$(shell pwd)
VERSION=0.0.0
MONOVA:=$(shell which monova dot 2> /dev/null)

version:
ifdef MONOVA
override VERSION=$(shell monova)
else
	$(info "Install monova (https://github.com/jsnjack/monova) to calculate version")
endif

bin/grm: bin/grm_linux_amd64
	cp bin/grm_linux_amd64 bin/grm

bin/grm_linux_amd64: version main.go cmd/*.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-X github.com/jsnjack/grm/cmd.Version=${VERSION}" -o bin/grm_linux_amd64

bin/grm_darwin_amd64: version main.go cmd/*.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-X github.com/jsnjack/grm/cmd.Version=${VERSION}" -o bin/grm_darwin_amd64

build: bin/grm bin/grm_linux_amd64 bin/grm_darwin_amd64

release: build
	tar --transform='s,_.*,,' --transform='s,bin/,,' -cz -f bin/grm_linux_amd64.tar.gz bin/grm_linux_amd64
	tar --transform='s,_.*,,' --transform='s,bin/,,' -cz -f bin/grm_darwin_amd64.tar.gz bin/grm_darwin_amd64
	grm release jsnjack/grm -f bin/grm -f bin/grm_linux_amd64.tar.gz -f bin/grm_darwin_amd64.tar.gz -t "v`monova`"

.PHONY: version release build
