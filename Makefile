.PHONY: all build clean

BINARY := earapi

all: build

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BINARY) .

clean:
	rm -f $(BINARY)
