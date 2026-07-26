.PHONY: all build linux clean

BINARY := earapi

all: build

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BINARY) .

linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY) .

clean:
	rm -f $(BINARY)
