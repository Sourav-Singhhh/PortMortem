# Port Mortem — High-Performance Picomatch Go Port
# Master Root Makefile for Evaluator & Contributor Commands

.PHONY: all build test test-original vet bench examples survivor fuzz clean help

# Default target
all: build test

## Help target listing available commands
help:
	@echo "Port Mortem — Master Makefile Commands:"
	@echo "  make build         — Build package module (cd port && go build ./...)"
	@echo "  make test          — Run full unit & differential test suite (cd port && go test -count=1 ./...)"
	@echo "  make test-original — Run unmodified original picomatch test suite against Go adapter"
	@echo "  make vet           — Perform static safety auditing (cd port && go vet ./...)"
	@echo "  make bench         — Execute 16-target performance benchmarks (cd port && go test -run=\"^$$\" -bench=\".\" -benchmem)"
	@echo "  make examples      — Execute GoDoc runnable example tests (cd port && go test -v -run=\"^Example\" .)"
	@echo "  make survivor      — Execute 60s differential fuzz survivor engine (cd port && go run ./fuzz_survivor -duration=60s)"
	@echo "  make fuzz          — Execute 10s parser fuzzing smoke test (cd port && go test -v -fuzz=FuzzCompile -fuzztime=10s .)"
	@echo "  make clean         — Invalidate build and test toolchain caches (cd port && go clean -cache -testcache)"

## Build package module
build:
	cd port && go build ./...

## Run unit, platform, unicode, normalization, ReDoS, and differential test suites
test:
	cd port && go test -count=1 ./...

## Run unmodified original picomatch test suite against Go adapter bridge
test-original:
	cd port && go build -o ../tests/go_adapter.exe ./cmd/adapter
	cd tests/original && NODE_PATH="../../original-picomatch/picomatch-master/node_modules" node "../../original-picomatch/picomatch-master/node_modules/mocha/bin/mocha.js" "*.js"


## Static safety and linter diagnostics
vet:
	cd port && go vet ./...

## Quantitative performance benchmarks
bench:
	cd port && go test -run="^$$" -bench="." -benchmem

## GoDoc runnable example tests
examples:
	cd port && go test -v -run="^Example" .

## Differential fuzz survivor engine against Node.js picomatch
survivor:
	cd port && go run ./fuzz_survivor -duration=60s

## Native Go parser fuzzing smoke test
fuzz:
	cd port && go test -v -fuzz=FuzzCompile -fuzztime=10s .

## Clean toolchain build and test caches
clean:
	cd port && go clean -cache -testcache
