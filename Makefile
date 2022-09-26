
.PHONY: clean
clean:
	rm -f remote-shell-client

.PHONY: compile
compile: clean
	go build -a -o remote-shell-client -ldflags="-s -w"
	stat remote-shell-client

.PHONY: tidy
tidy:
	go mod verify
	go mod tidy
	@if ! git diff --quiet go.mod go.sum; then \
		echo "please run go mod tidy and check in changes, you might have to use the same version of Go as the CI"; \
		exit 1; \
	fi

.PHONY: lint-install
lint-install:
	@echo "Installing golangci-lint"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.46.2

.PHONY: lint
lint:
	@which golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not found, please run: make lint-install"; \
		exit 1; \
	}
	golangci-lint run

.PHONY: test-release
test-release:
	goreleaser release --skip-publish --rm-dist --snapshot