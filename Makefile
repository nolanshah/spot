
format:
	gofmt -s -w ./

build: 
	go build

release:
	goreleaser release --snapshot --clean

test-local:
	go run main --input test/input --output test/output