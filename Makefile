BINARY := menubar
PKG := ./cmd/menubar

.PHONY: all build run test vet fmt tidy clean

all: build

build:
	go build -o $(BINARY) $(PKG)

run: build
	./$(BINARY)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

clean:
	rm -f $(BINARY)
