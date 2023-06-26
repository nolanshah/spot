.PHONY: format build release test-transform test-watch clean install-deps-mac

format:
	gofmt -s -w ./

build:
	go build -o build/bloop

release:
	goreleaser release --snapshot --clean

clean:
	rm -rf test/output/*
	rm -rf build
	rm -rf dist

install-deps-mac:
	brew install golang
	brew install goreleaser/tap/goreleaser
	brew instlal pandoc
