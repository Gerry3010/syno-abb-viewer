BIN = syno-abb-viewer
PKG = ./cmd/syno-abb-viewer

.PHONY: build run test vet tidy clean

build:
	go build -o $(BIN) $(PKG)

run:
	go run $(PKG)

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -f $(BIN)
