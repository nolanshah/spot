
format:
	gofmt -s -w ./

build: 
	go build

release:
	goreleaser release --snapshot --clean

test-transform:
	go run main --input test/input --output test/output --debug

test-watch:
	go run main --input test/input --output test/output --debug --watch --addr :8081

clean:
	rm -r test/output/*
	rm main
	rm -r dist
