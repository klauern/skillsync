# Build variables
BINARY_NAME := "skillsync"
VERSION := env("VERSION", "0.1.0")
BUILD_DIR := "bin"
COMMIT := shell("git rev-parse --short HEAD")
BUILD_DATE := datetime_utc("%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := "-ldflags \"-X github.com/klauern/skillsync/internal/cli.Version=" + VERSION + " -X github.com/klauern/skillsync/internal/cli.Commit=" + COMMIT + " -X github.com/klauern/skillsync/internal/cli.BuildDate=" + BUILD_DATE + "\""

[group("help"), doc("List available recipes")]
default:
  @just --list

[group("help"), doc("Alias for default help output")]
help: default

[group("build"), doc("Build the binary")]
build:
  @mkdir -p {{BUILD_DIR}}
  go build {{LDFLAGS}} -o {{BUILD_DIR}}/{{BINARY_NAME}} ./cmd/skillsync

[group("build"), doc("Install the binary to GOPATH/bin")]
install:
  go install {{LDFLAGS}} ./cmd/skillsync

[group("build"), doc("Remove installed binary from GOPATH/bin")]
uninstall:
  bin_dir="${GOBIN:-$(go env GOPATH)/bin}"; \
  rm -f "${bin_dir}/{{BINARY_NAME}}$(go env GOEXE)"

[group("test"), doc("Run tests with race and coverage")]
test:
  go test -v -race -coverprofile=coverage.out ./...

[group("test"), doc("Run tests and open coverage report")]
test-coverage: test
  go tool cover -html=coverage.out

[group("quality"), doc("Run golangci-lint")]
lint:
  golangci-lint run ./...

[group("quality"), doc("Format code with gofumpt and goimports")]
fmt:
  gofumpt -w .
  goimports -w -local github.com/klauern/skillsync .

[group("quality"), doc("Run go vet")]
vet:
  go vet ./...

[group("quality"), doc("Tidy and verify modules")]
tidy:
  go mod tidy
  go mod verify

[group("tools"), doc("Install gofumpt, goimports, and golangci-lint")]
install-tools:
  @echo "Installing gofumpt..."
  @go install mvdan.cc/gofumpt@latest
  @echo "Installing goimports..."
  @go install golang.org/x/tools/cmd/goimports@latest
  @echo "Installing golangci-lint..."
  @go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
  @echo "All tools installed successfully!"

[group("cleanup"), doc("Remove build artifacts and coverage output")]
clean:
  rm -rf {{BUILD_DIR}}
  rm -f coverage.out coverage.html

[group("quality"), doc("Run all quality checks")]
audit: tidy fmt vet lint test

[group("build"), doc("Build and run the binary")]
run: build
  ./{{BUILD_DIR}}/{{BINARY_NAME}}

[group("meta"), doc("Run audit and build")]
all: audit build
