GOROOT_WASM := $(shell f="$$(go env GOROOT)/lib/wasm/wasm_exec.js"; [ -f "$$f" ] || f="$$(go env GOROOT)/misc/wasm/wasm_exec.js"; echo $$f)

.PHONY: all test web bundle serve clean

all: test web bundle

test:
	go test ./...

# web/: index.html + main.wasm + wasm_exec.js + vendor/ -> upload the folder to your blog
web:
	GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o web/main.wasm ./wasm
	cp "$(GOROOT_WASM)" web/wasm_exec.js

# dist/thompson-nfa.html: one file with everything inlined
bundle: web
	go run ./cmd/bundle -o dist/thompson-nfa.html

serve: web
	@echo "http://localhost:8080"
	cd web && python3 -m http.server 8080

clean:
	rm -rf dist web/main.wasm web/wasm_exec.js
