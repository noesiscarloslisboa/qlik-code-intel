.PHONY: build test integration vet format-check generate test-grammar check-generated check benchmark

build:
	go build -o bin/qlik-repomap ./cmd/qlik-repomap

test:
	go test -race ./...

integration:
	go test -tags=integration ./...

vet:
	go vet ./...

format-check:
	test -z "$$(gofmt -l cmd internal tree-sitter-qlik/bindings/go)"

generate:
	cd tree-sitter-qlik && npm run generate
	go clean -cache

test-grammar:
	cd tree-sitter-qlik && npm test

check-generated:
	cd tree-sitter-qlik && npm run check-generated

check: check-generated test-grammar format-check test integration vet build

# Optional source-reviewed CLI benchmark; requires Python 3.9+ (stdlib only).
benchmark: build
	python3 -m unittest discover -s scripts -p 'test_benchmark_retrieval.py'
	python3 scripts/benchmark-retrieval.py --samples-dir testdata/retrieval --manifest testdata/retrieval/questions.json
