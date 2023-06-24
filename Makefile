
format:
	gofmt -s -w ./

build: 
	go build -o build/bloop

release:
	goreleaser release --snapshot --clean

test-transform:
	go run main --input test/input --output test/output --debug

test-watch:
	go run main --input test/input --output test/output --debug --watch --addr :8081

clean:
	rm -rf test/output/*
	rm -rf build
	rm -rf dist
