# Upstream Sync Guide

本文档记录了基于开源项目 [new-api](https://github.com/Calcium-Ion/new-api) 所做的自定义修改，方便在上游更新时进行代码合并和冲突处理。

当前基于上游版本：`ebbe3155`（`main` 分支）

---

## 修改清单

### 1. Dockerfile — 添加 Go 代理

**文件**：`Dockerfile`

**改动**：在构建阶段添加 `GOPROXY` 环境变量，加速国内依赖下载。

```dockerfile
ENV GOPROXY=https://goproxy.cn,direct
```

**同步注意**：合并时保留此行即可，无冲突风险。

---

### 2. controller/user.go — 注册接口返回用户 ID

**文件**：`controller/user.go`

**改动**：`Register` 函数注册成功后，在响应中返回新用户的 `id`。

```go
c.JSON(http.StatusOK, gin.H{
    "success": true,
    "message": "",
    "data": gin.H{
        "id": insertedUser.Id,
    },
})
```

**用途**：modelgate 后端在创建新用户后需要立即获取用户 ID，用于后续的 new-api Token 创建等操作。

**同步注意**：如果上游修改了 `Register` 函数的响应格式，需要保留 `data.id` 字段。

---

### 3. router/api-router.go — 注册管理员 Token 管理路由

**文件**：`router/api-router.go`

**改动**：在 API 路由组中调用 `RegisterAdminTokenRoutes`，注册 `/admin/token/*` 系列路由。

```go
// Admin token 管理（供 modelgate 后端调用）
RegisterAdminTokenRoutes(apiRouter)
```

**同步注意**：此行添加在 `tokenRoute` 路由组之后、`usageRoute` 路由组之前，合并时注意保留。

---

### 4. 新增文件：controller/admin_token.go

**文件**：`controller/admin_token.go`（新增）

**功能**：管理员 Token 操作控制器，供 modelgate 后端服务调用。包含以下接口：

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| AdminCreateToken | POST | `/admin/token/create` | 为指定用户创建 API Token，返回完整 key |
| AdminDeleteToken | POST | `/admin/token/delete` | 删除指定 Token |
| AdminUpdateTokenStatus | POST | `/admin/token/status` | 更新 Token 状态（启用/禁用） |
| AdminUpdateTokenQuota | POST | `/admin/token/quota` | 更新 Token 配额（remain_quota / unlimited_quota） |
| AdminGetTokenInfo | POST | `/admin/token/info` | 查询指定 Token 的详细信息（含配额） |

**请求/响应格式**：

- `AdminCreateToken`：接收 `user_id`、`name`、可选的 `remain_quota` 和 `unlimited_quota`，返回 `id` 和 `key`
- `AdminDeleteToken`：接收 `token_id`
- `AdminUpdateTokenStatus`：接收 `token_id` 和 `status`（1=启用, 2=禁用）
- `AdminUpdateTokenQuota`：接收 `token_id` 和可选的 `remain_quota`、`unlimited_quota`
- `AdminGetTokenInfo`：接收 `token_id`，返回 Token 完整信息

**同步注意**：此文件为全新文件，上游不存在，不会产生合并冲突。但如果上游修改了 `model.Token` 结构体，可能需要适配。

---

### 5. 新增文件：middleware/service_auth.go

**文件**：`middleware/service_auth.go`（新增）

**功能**：服务间调用认证中间件 `ServiceAuth()`，用于验证 modelgate 等后端服务对 new-api 的管理接口调用。

**认证流程**：
1. 从 `Authorization` header 提取 Bearer access_token
2. 调用 `model.ValidateAccessToken` 验证 token 有效性
3. 校验用户角色为管理员（`RoleAdminUser` 及以上）
4. 校验用户状态未被禁用
5. 将 `username`、`role`、`id` 写入 gin context

**与现有 `auth` 中间件的区别**：不走 session 认证，仅验证 access_token + 管理员权限，适用于服务间 API 调用场景。

**同步注意**：此文件为全新文件，上游不存在，不会产生合并冲突。但如果上游修改了 `model.ValidateAccessToken` 或用户角色/状态的常量定义，可能需要适配。

---

### 6. 新增文件：router/admin_token_routes.go

**文件**：`router/admin_token_routes.go`（新增）

**功能**：注册管理员 Token 管理路由，所有路由使用 `ServiceAuth()` 中间件进行认证。

**路由组**：`/admin/token`，包含：

```
POST /admin/token/create  → controller.AdminCreateToken
POST /admin/token/delete  → controller.AdminDeleteToken
POST /admin/token/status  → controller.AdminUpdateTokenStatus
POST /admin/token/quota   → controller.AdminUpdateTokenQuota
POST /admin/token/info    → controller.AdminGetTokenInfo
```

**同步注意**：此文件为全新文件，上游不存在，不会产生合并冲突。

---

## 合并策略建议

1. **低风险**：`Dockerfile`（GOPROXY）、新增的三个文件（`admin_token.go`、`service_auth.go`、`admin_token_routes.go`）—— 直接保留即可
2. **中风险**：`router/api-router.go` 中的 `RegisterAdminTokenRoutes` 调用——注意保留插入位置
3. **中风险**：`controller/user.go` 中注册接口返回 `data.id`——注意与上游 `Register` 函数变更合并
4. **需关注**：如果上游修改了 `model.Token`、`model.User` 结构体或认证相关函数，需要检查新增的控制器和中间件是否需要适配
