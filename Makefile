all: deps build

build: build-dmrstream

build-dmrstream:
	go build ./cmd/dmrstream/
	@ls -alh dmrstream

deps: godeps oggfwd

godeps:
	go get -v $(shell go list -f '{{ join .Deps "\n" }}' ./... | sort -u | egrep '(gopkg|github)' | grep -v '/tehmaze/go-dmr')

oggfwd:
	$(CC) -O2 -pipe -Wall -ffast-math -fsigned-char -lshout -pthread -o $@ $@.c
