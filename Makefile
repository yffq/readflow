.PHONY: build run test clean build-all docker-push

BINARY=readflow
BIN_DIR=bin

# --- 阿里云容器镜像服务 ---
# 先去 https://cr.console.aliyun.com/ 创建个人实例 → 命名空间（如 readflow）
# 然后改下面这个变量：
ACR_NAMESPACE=your-namespace
ACR_REGISTRY=your-registry.aliyuncs.com
IMAGE=$(ACR_REGISTRY)/$(ACR_NAMESPACE)/readflow:latest

build:
	CGO_ENABLED=0 go build -o $(BIN_DIR)/$(BINARY) ./cmd/server

build-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BIN_DIR)/$(BINARY)-linux-amd64 ./cmd/server

build-all: build build-linux-amd64

run: build
	./$(BIN_DIR)/$(BINARY)

test:
	go test ./... -count=1

vet:
	go vet ./...

clean:
	rm -rf $(BIN_DIR)/ data/

# --- Docker / ACR ---
acr-login:
	docker login --username=@你的阿里云账号 $(ACR_REGISTRY)

docker-build:
	docker build --platform linux/amd64 -t readflow:latest .

docker-push: docker-build
	docker tag readflow:latest $(IMAGE)
	docker push $(IMAGE)
	@echo "Pushed: $(IMAGE)"
