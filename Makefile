all: deps build

build: build-dmrstream

build-dmrstream:
	go build ./cmd/dmrstream/
	@ls -alh dmrstream

deps: godeps deps-platform oggfwd

deps-platform:
	@echo "For OS X:     make deps-brew"
	@echo "For Debian:   make deps-debian"

deps-brew:
	brew install --HEAD mbelib
	brew install lame
	brew install sox

deps-debian:
	sudo apt-get install lame sox

godeps:
	go get -v $(shell go list -f '{{ join .Deps "\n" }}' ./... | sort -u | egrep '(gopkg|github)' | grep -v '/tehmaze/go-dmr')

oggfwd:
	$(CC) -O2 -pipe -Wall -ffast-math -fsigned-char -lshout -pthread -o $@ $@.c
