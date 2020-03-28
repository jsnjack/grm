BINARY:=grm
PWD:=$(shell pwd)
VERSION=0.0.0
MONOVA:=$(shell which monova dot 2> /dev/null)

version:
ifdef MONOVA
override VERSION=$(shell monova)
else
	$(info "Install monova (https://github.com/jsnjack/monova) to calculate version")
endif

.ONESHELL:
bin/${BINARY}: version main.go cmd/*.go
	go build -ldflags="-X main.Version=${VERSION}" -o bin/${BINARY}

build: bin/${BINARY}

release: build
	python ~/lxdfs/cobro/ci/utils/release_on_github.py -f bin/${BINARY} -r jsnjack/grm -t "v`monova`"

.PHONY: version
