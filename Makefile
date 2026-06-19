.PHONY: build install deps clean

build:
	go build -o fall .

install: build
	cp fall ~/bin/fall
	cp scripts/fall-index ~/bin/fall-index
	cp scripts/fall-serve ~/bin/fall-serve

deps:
	go install github.com/sourcegraph/zoekt/cmd/zoekt-git-index@latest
	go install github.com/sourcegraph/zoekt/cmd/zoekt-webserver@latest

clean:
	rm -f fall
