format:
	go fmt ./...

lint:
	go vet ./...

test:
	go test ./...

build-linux:
	go build -o ./build/merklediff-linux-amd64 ./cmd/merklediff

build-mac:
	go build -o ./build/merklediff-darwin-amd64 ./cmd/merklediff

build-windows:
	go build -o ./build/merklediff-windows-amd64.exe ./cmd/merklediff

build-all: build-linux build-mac build-windows

run:
	./build/merklediff