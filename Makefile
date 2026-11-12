.PHONY: all api build linux clean ui health-ui tools/tailwindcss vendor-daisyui

BINARY := earapi
ESBUILD ?= esbuild
TAILWINDCSS ?= ./tools/tailwindcss
DAISYUI_VERSION ?= 5.0.50
UI_SRC := health/ui-src
UI_OUT := health/ui
LDFLAGS := -s -w

# Default: compile all embedded UIs, then link earapi (ready to run).
all: api

ui: health-ui

api: ui
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BINARY) .

build: api

linux: ui
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BINARY) .

health-ui: tools/tailwindcss
	@mkdir -p $(UI_OUT)
	$(ESBUILD) $(UI_SRC)/main.js --bundle --minify --format=iife --outfile=$(UI_OUT)/app.js
	$(TAILWINDCSS) -i $(UI_SRC)/app.css -o $(UI_OUT)/app.css --minify
	cp $(UI_SRC)/index.html $(UI_OUT)/index.html

tools/tailwindcss:
	@mkdir -p tools
	@if [ ! -x tools/tailwindcss ]; then \
	  echo "Downloading tailwindcss v4.1.8 (linux-x64)…"; \
	  curl -fsSL "https://github.com/tailwindlabs/tailwindcss/releases/download/v4.1.8/tailwindcss-linux-x64" -o tools/tailwindcss; \
	  chmod +x tools/tailwindcss; \
	fi

vendor-daisyui:
	@echo "Vendoring daisyui $(DAISYUI_VERSION)…"
	@mkdir -p $(UI_SRC)/vendor
	curl -fsSL "https://registry.npmjs.org/daisyui/-/daisyui-$(DAISYUI_VERSION).tgz" -o /tmp/daisyui.tgz
	rm -rf $(UI_SRC)/vendor/daisyui
	mkdir -p $(UI_SRC)/vendor/daisyui
	tar -xzf /tmp/daisyui.tgz -C $(UI_SRC)/vendor/daisyui --strip-components=1

clean:
	rm -f $(BINARY)
	rm -rf $(UI_OUT)/app.js $(UI_OUT)/app.css $(UI_OUT)/index.html
