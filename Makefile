BINARY     := bin/one
CMD        := ./cmd/one
MODULE     := elydelva/one
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS    := -ldflags "-X main.version=$(VERSION)"

.PHONY: build build-nowasm install test test-security test-e2e bench lint clean release tidy

build:
	@mkdir -p bin
	go build $(LDFLAGS) -o $(BINARY) $(CMD)

build-nowasm:
	@mkdir -p bin
	go build -tags=nowasm $(LDFLAGS) -o $(BINARY)-nowasm $(CMD)

install:
	go install $(LDFLAGS) $(CMD)

test:
	go test ./...

test-security:
	go test -tags=security ./tests/security/...

test-e2e:
	go test -tags=e2e ./tests/e2e/...

bench:
	go test -bench=. -benchmem -run='^$$' ./... 2>&1 | tee /tmp/bench.txt
	@go run ./scripts/bench_check.go /tmp/bench.txt .benchmarks.json 2>/dev/null || true

lint:
	golangci-lint run

clean:
	rm -rf bin/ dist/ coverage.out

release:
	goreleaser release --clean

tidy:
	go mod tidy

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
