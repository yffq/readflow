# Readflow 项目原则

- 小程序端和网页端的页面结构、功能、交互尽量保持一致
- Go 后端代码不要静默忽略错误（不用 `_` 吞掉 error）
- 不暴露个人信息（域名、密钥等）到代码中
- 扩展和第三方的 URL 配置使用空字符串作为默认值
- 提交信息使用英文，格式：`type: description`

## 常用命令

- `go test ./...` — 运行所有测试
- `go build ./...` — 编译检查
- `make docker-push` — 构建并推送 Docker 镜像
- `go run ./cmd/server` — 本地启动开发服务器
