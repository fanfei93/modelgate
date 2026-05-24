.PHONY: run dev-web build-web build all clean

# 启动后端（开发模式，go run）
run:
	go run main.go

# 启动前端 dev server（Vite 热更新）
dev-web:
	cd web && npm run dev

# 构建前端产物 -> web/dist/
build-web:
	cd web && npm install && npm run build

# 构建后端 binary
build:
	go build -o modelgate-server .

# 完整生产构建：前端 + 后端
all: build-web build

# 清理
clean:
	rm -f modelgate-server
	rm -rf web/dist
