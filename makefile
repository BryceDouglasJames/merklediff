format:
	go fmt ./...

lint:
	go vet ./...

test:
	go test ./...

build-linux:
	go build -o ./build/linux/merklediff ./cmd/merklediff

build-mac:
	go build -o ./build/macos/merklediff ./cmd/merklediff

build-windows:
	go build -o ./build/windows/merklediff.exe ./cmd/merklediff

build-all: build-linux build-mac build-windows

run:
	go run ./cmd/merklediff