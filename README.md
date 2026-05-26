# ModelGate

团队级 API 调用配额管理服务。

## 本地部署

### 前置要求

- Go 1.21+
- Node.js 18+
- MySQL 5.7+

### 步骤

**1. 克隆项目**

```bash
git clone https://github.com/modelgate/modelgate.git
cd modelgate
```

**2. 配置数据库**

创建 MySQL 数据库：

```bash
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS modelgate CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
```

修改 `config.yaml` 中的数据库连接信息：

```yaml
database:
  driver: mysql
  dsn: "用户名:密码@tcp(localhost:3306)/modelgate?charset=utf8mb4&parseTime=True&loc=Local"
```

**3. 启动 new-api 服务**

本服务依赖 `new-api`，需要先启动：

```bash
cd new-api
# 参考 new-api/README.md 完成部署
```

确保 `config.yaml` 中的 new_api 配置正确指向 new-api 服务地址。

**4. 启动后端**

```bash
go run main.go
# 或者使用 make
make run
```

服务将在 `http://localhost:8080` 启动。

**5. 启动前端（可选）**

若只需测试 API，可跳过此步。若需完整 Web 界面：

```bash
cd web
npm install
npm run dev
```

前端开发服务器默认在 `http://localhost:5173`。

**6. 构建生产版本**

```bash
# 构建前端产物
make build-web

# 构建后端
make build

# 一步到位
make all
```

### Docker 部署

**一键启动所有服务：**

```bash
docker-compose up -d
```

这将启动：
- MySQL (localhost:3306)
- new-api (localhost:3000)
- ModelGate 后端 + 前端 (localhost:8080)

**查看日志：**

```bash
docker-compose logs -f
```

**停止服务：**

```bash
docker-compose down
```

**注意事项：**
- 首次启动后，new-api 需要在管理后台创建管理员账号并获取 admin_key
- 获取到 admin_key 后，需要更新 `config.docker.yaml` 中的 `new_api.admin_key` 并重启：
  ```bash
  docker-compose restart modelgate
  ```
