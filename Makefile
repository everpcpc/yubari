mkfile_path	:= $(abspath $(lastword $(MAKEFILE_LIST)))
current_dir	:= $(patsubst %/,%,$(dir $(mkfile_path)))

export GOPATH := $(current_dir)/go

default: dep build

gopath:
	@if [ ! -d "go" ]; then mkdir go; fi
	@if [ ! -d "go/bin" ]; then mkdir go/bin; fi

dep: gopath
	@echo GOPATH=${GOPATH}
	@echo "Installing depends..."
	@go get -v github.com/op/go-logging

build:
	@echo "Building..."
	@(cd yubari; go build)
	@mv ${current_dir}/yubari/yubari ${GOPATH}/bin/
