# The desktop dev loop is cgo-free: the toolkit reaches C through purego.
DEV := CGO_ENABLED=0
.DEFAULT_GOAL := help
.PHONY: help build test check check-nocgo test-visual parity-web examples run fmt-check

help: ## list the targets
	@grep -hE '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | sed 's/:.*## /\t/' | expand -t24

build: ## build the toolkit, examples, harness and tools
	$(DEV) go build ./...

test: ## the headless test suites (no display needed)
	$(DEV) go test ./...

check: build test check-nocgo ## the gate: build + tests + the cgo guard

check-nocgo: ## fail if cgo creeps into the desktop dependency graph
	@sh scripts/check-nocgo.sh

test-visual: ## real-window visual suite; needs a GL display, cgo-free on purpose
	$(DEV) go test -count=1 -tags visual ./harness/behaviour/...

parity-web: ## render the parity scene offscreen and through GL, then diff them
	$(DEV) go run ./cmd/tools/qmlshot -preset web-light -w 800 -h 600 -o /tmp/meui_parity_soft.png examples/parity/parity.qml
	$(DEV) go run ./cmd/tools/glshot  -preset web-light -w 800 -h 600 -o /tmp/meui_parity_gl.png   examples/parity/parity.qml
	$(DEV) go run ./cmd/tools/imgdiff /tmp/meui_parity_soft.png /tmp/meui_parity_gl.png

examples: ## the examples launcher
	$(DEV) go run ./examples/launcher

run: ## the imported QML demo (the kitchen sink)
	$(DEV) go run ./examples/enterprise

fmt-check: ## fail if any Go file is not gofmt-ed
	@out=$$(gofmt -l $$(find . -name '*.go' -not -path './.git/*')); \
	[ -z "$$out" ] || { echo "gofmt needed:"; echo "$$out"; exit 1; }
	@echo "ok: gofmt clean"
