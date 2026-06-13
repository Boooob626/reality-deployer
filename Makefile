# Reality Deployer — build / test / lint
# 交叉编译产物落在 dist/，VPS 上无需安装 Go 工具链。

APP      := reality-deployer
PKG      := ./...
BIN_DIR  := dist

GOFLAGS  := -trimpath -ldflags="-s -w"

## 公共目标 ##

.PHONY: all
all: lint test build

.PHONY: build
build: ## 编译当前平台二进制到 dist/$(APP)
	mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -o $(BIN_DIR)/$(APP) ./cmd/deploy

.PHONY: build-all
build-all: ## 交叉编译 VPS 常见架构（amd64 / arm64）
	mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o $(BIN_DIR)/$(APP)-linux-amd64 ./cmd/deploy
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -o $(BIN_DIR)/$(APP)-linux-arm64 ./cmd/deploy

.PHONY: vet
vet: ## go vet
	go vet $(PKG)

.PHONY: test
test: ## 运行单元测试
	go test $(PKG)

.PHONY: lint
lint: vet ## 静态检查（vet 为底线；装了 golangci-lint 则一并跑）
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || echo "[lint] golangci-lint 未安装，跳过（已执行 go vet）"

.PHONY: shellcheck
shellcheck: ## 检查所有 shell 脚本
	@command -v shellcheck >/dev/null 2>&1 && shellcheck deploy.sh scripts/*.sh scripts/lib/*.sh || echo "[shellcheck] 未安装，跳过"

.PHONY: clean
clean:
	rm -rf $(BIN_DIR) staging manifest.json plan.sh apply.env

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-14s %s\n", $$1, $$2}'
