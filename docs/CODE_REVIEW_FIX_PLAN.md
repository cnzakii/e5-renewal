# E5 Renewal 代码审查修复方案

## Context

基于对整个项目的全面代码审查，发现 2 个严重、5 个高危、9+ 个中等问题。本文档既是修复方案参考文档（沉淀到项目中），也是可执行的分批实施计划。

## 实施约束

1. **SQLite 驱动不可替换**: 必须使用 `github.com/glebarez/sqlite`（纯 Go），禁止替换为 `gorm.io/driver/sqlite` 等需要 CGO 的版本。项目使用 distroless 镜像，必须保证静态编译、最小体积。
2. **所有代码注释必须使用英文**: 包括新增和修改的注释、godoc、JSDoc、TODO 等，一律用英文书写。

## 实施策略

分 4 个迭代批次，按优先级从高到低执行。每批完成后运行完整测试验证。

---

## 批次 1：安全关键修复（Critical + High 安全问题）

### 1.1 XSS 注入 — `pathPrefix` 未转义
- **文件**: `backend/spa/handler.go:61`
- **问题**: `pathPrefix` 直接拼接到 `<script>` 标签，环境变量可注入 XSS
- **修复**: 使用 `json.Marshal` 对 `pathPrefix` 进行安全编码

```go
// handler.go buildIndexHTML() 第 61 行
// 修复前:
injection := `<script>window.__E5_CONFIG__={"pathPrefix":"` + pathPrefix + `"}</script>`
// 修复后:
safePrefix, _ := json.Marshal(pathPrefix)
injection := `<script>window.__E5_CONFIG__={"pathPrefix":` + string(safePrefix) + `}</script>`
```
需要在 import 中添加 `"encoding/json"`。

### 1.2 `math/rand` 并发数据竞争
- **文件**: `backend/main.go:36`
- **问题**: 共享的 `*rand.Rand` 在多 goroutine 间不安全使用
- **修复**: 为 Executor 和 Scheduler 各创建独立的 `*rand.Rand`

```go
// main.go 第 36-39 行
// 修复前:
rng := rand.New(rand.NewSource(time.Now().UnixNano()))
exec := executor.New(oauthSvc, rng)
sched := scheduler.New(exec, rng)
// 修复后:
execRng := rand.New(rand.NewSource(time.Now().UnixNano()))
schedRng := rand.New(rand.NewSource(time.Now().UnixNano() + 1))
exec := executor.New(oauthSvc, execRng)
sched := scheduler.New(exec, schedRng)
```

### 1.3 OAuth `postMessage` 使用 `'*'` 广播 token
- **文件**: `backend/handlers/oauth.go:111`
- **问题**: OAuth token 广播给任意窗口
- **修复**: 将 `prefix` 参数传入 `oauthResultHTML` 并构造正确的 origin；同时在前端验证 `e.origin`

后端 `oauth.go`:
```go
// oauthResultHTML 函数签名改为接收 origin
func oauthResultHTML(status, payload, origin string) []byte {
    safeOrigin, _ := json.Marshal(origin)
    // ...
    // 第 111 行修改:
    // 修复前: window.opener.postMessage({ type: 'e5-oauth-result', data: result }, '*');
    // 修复后: window.opener.postMessage({ type: 'e5-oauth-result', data: result }, %s);
    // 其中 %s 为 safeOrigin
```

回调 handler 需传入 origin:
```go
// 构造 origin: scheme + "://" + host
scheme := "http"
if c.Request.TLS != nil {
    scheme = "https"
}
origin := fmt.Sprintf("%s://%s", scheme, c.Request.Host)
```

前端 `AccountFormDialog.vue` 的 postMessage 监听器添加 origin 校验:
```typescript
// 在 oauthListener 回调开头添加:
if (e.origin !== window.location.origin) return
```

### 1.4 `client_secret` 通过 GET 查询参数传递
- **文件**: `backend/handlers/oauth.go:61-63`
- **问题**: GET 参数记录到日志、浏览器历史、代理
- **修复**: 改 `/authorize` 为 POST 方法，从请求体读取

```go
// 修复前: authGroup.GET("/authorize", func(c *gin.Context) {
// 修复后: authGroup.POST("/authorize", func(c *gin.Context) {
//   var req struct {
//       ClientID     string `json:"client_id" binding:"required"`
//       ClientSecret string `json:"client_secret"`
//       TenantID     string `json:"tenant_id" binding:"required"`
//   }
//   if err := c.ShouldBindJSON(&req); err != nil { ... }
```

前端 `AccountFormDialog.vue` 对应的 API 调用也需从 GET 改为 POST。

### 1.5 前端无 401 响应拦截器
- **文件**: `frontend/src/api/client.ts`
- **问题**: token 过期后用户看到空白页面
- **修复**: 添加 response interceptor

```typescript
// client.ts 追加:
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      const { clearAuth } = useAuth()
      clearAuth()
      window.location.href = `${pathPrefix}/login`
    }
    return Promise.reject(error)
  }
)
```

### 1.6 前端 postMessage 监听未校验 origin
- **文件**: `frontend/src/components/AccountFormDialog.vue:590`
- **修复**: 见 1.3，在监听回调开头添加 `if (e.origin !== window.location.origin) return`

---

## 批次 2：并发与资源管理修复（High）

### ~~2.1 Scheduler WaitGroup 竞态条件~~ — 已移除（误报）
> 经复查，当前代码 `t.Stop()` 返回 `true` 时才调用 `wg.Done()` 补偿，是正确的 Go 惯用模式，不存在 double-Done panic。

### 2.2 通知配置双重加载
- **文件**: `backend/services/scheduler/scheduler.go:286-356`
- **问题**: `checkAuthExpiry` 和 `notifyIfEnabled` 分别加载配置，存在 TOCTOU
- **修复**: `notifyIfEnabled` 接收已加载的 config，不再重复加载

```go
func (s *Scheduler) notifyIfEnabled(cfg *models.NotificationConfig, eventKey, title, message string) {
    if s.Notifier == nil || cfg.URL == "" {
        return
    }
    // ... 使用传入的 cfg 判断 enabled
}
```

所有调用点传入已加载的 `cfg`。

---

## 批次 3：中等问题修复

### 3.1 使用 `gin.New()` 替代 `gin.Default()`
- **文件**: `backend/main.go:41`
- **修复**: 使用 `gin.New()` + 显式注册自定义中间件

```go
r := gin.New()
r.Use(middleware.SlogLogger(), middleware.SlogRecovery())
```

### 3.2 前端路由缺少 catch-all 和根路径重定向
- **文件**: `frontend/src/router/index.ts`
- **修复**:

```typescript
return [
    { path: '/login', component: LoginView, meta: { guest: true } },
    { path: '/', redirect: '/dashboard' },  // 新增
    {
      path: '/',
      component: AppLayout,
      children: [
        { path: '/dashboard', component: DashboardView },
        // ...
      ],
    },
    { path: '/:pathMatch(.*)*', redirect: '/dashboard' },  // 新增 catch-all
]
```

### 3.3 速率限制器 IP 伪造
- **文件**: `backend/main.go`
- **修复**: 添加可信代理配置

```go
r := gin.New()
r.SetTrustedProxies(nil) // 或从配置读取
```

### ~~3.4 数据库目录权限~~ — 已移除（误报）
> 经复查，`0755` 对目录是标准权限（rwxr-xr-x），不算过于宽松。

### 3.5 HTTP 连接复用受损
- **文件**: `backend/services/graph/caller.go:260-263`
- **修复**: 读取完 body 后 drain

```go
bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
io.Copy(io.Discard, resp.Body) // drain remaining
```

### 3.6 `DeleteCascade` 忽略 Pluck 错误
- **文件**: `backend/database/account.go:67-84`
- **修复**: 检查 Pluck 返回的错误

```go
if err := tx.Model(&models.TaskLog{}).Where("account_id = ?", id).Pluck("id", &logIDs).Error; err != nil {
    return err
}
```

### 3.7 ConfirmDialog 默认按钮文本硬编码中文
- **文件**: `frontend/src/components/ConfirmDialog.vue:69-72`
- **问题**: 默认值 `'确定'` / `'取消'` 硬编码中文，i18n 中已有 `confirm.ok` / `confirm.cancel` 两个 key
- **修复**: 使用已有 i18n key

```typescript
import { useI18n } from '../i18n'
const { t } = useI18n()
// withDefaults 中:
confirmText: t('confirm.ok'),
cancelText: t('confirm.cancel'),
```

### 3.8 添加缺失的 i18n key
- **文件**: `frontend/src/i18n/index.ts`
- 添加以下缺失 key 到 zh 和 en:
  - `accounts.form.refreshToken.oauth.parseError`
  - `accounts.form.refreshToken.oauth.failed`

### 3.9 DashboardView statCards 响应性问题
- **文件**: `frontend/src/views/DashboardView.vue:483-486`
- **问题**: `computed(() => ...).value` 立即求值，丢失响应性
- **修复**: 将 `valueClass` 改为函数调用

```typescript
// 修复前:
valueClass: computed(() => data.value.error_count > 0 ? 'text-red-500' : '...').value
// 修复后:
valueClass: () => data.value.error_count > 0 ? 'text-red-500' : '...'
```
模板中对应改为 `:class="card.valueClass?.()"`.

### 3.10 AppSidebar isDark 不响应系统主题变化
- **文件**: `frontend/src/components/AppSidebar.vue:137`
- **修复**: 使用 MutationObserver 监听 class 变化

```typescript
const isDark = ref(document.documentElement.classList.contains('dark'))
const observer = new MutationObserver(() => {
  isDark.value = document.documentElement.classList.contains('dark')
})
onMounted(() => observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] }))
onUnmounted(() => observer.disconnect())
```

### 3.11 将所有中文代码注释改为英文
- **涉及文件**:
  - `backend/spa/handler.go` — 6 处中文注释
  - `backend/handlers/oauth.go` — 3 处
  - `backend/services/executor/executor.go` — 4 处
  - `backend/services/scheduler/scheduler.go` — 1 处
  - `backend/services/oauth/state.go` — 3 处
  - `backend/services/graph/caller.go` — 1 处
  - `backend/middleware/ratelimit.go` — 1 处
  - `backend/models/models.go` — 1 处
  - `backend/services/security/crypto_test.go` — 1 处
  - `backend/handlers/account_test.go` — 5 处
  - `backend/services/oauth/state_test.go` — 1 处
  - `frontend/src/main.ts` — 2 处
  - `frontend/src/config.ts` — 1 处
  - `frontend/src/views/AccountsView.vue` — 1 处
  - `frontend/src/components/AccountFormDialog.vue` — 2 处
- **修复**: 逐一翻译为等义英文注释。测试文件中引用 i18n 中文文本的注释保留原文引用但用英文描述。

### 3.12 Vite 构建 base 改为相对路径，移除 handler.go 中的字符串替换 hack
- **文件**: `frontend/vite.config.ts`, `backend/spa/handler.go:64-68`
- **问题**: 当前 Vite `base` 默认为 `"/"`，构建产物中资源路径为绝对路径（`/assets/xxx.js`）。后端 `handler.go` 用 `bytes.ReplaceAll` 给 index.html 中的 `"/assets/` 和 `"/favicon` 加上 pathPrefix，但这种方式脆弱且不完整：
  - 只覆盖了 `"/assets/` 和 `"/favicon`，新增的资源路径会被遗漏
  - CSS 文件内部的 `url()` 引用（如字体、图片）不会被替换
  - 如果 Vite 构建输出结构变化，替换逻辑会静默失效
- **修复**:
  1. `vite.config.ts` 添加 `base: "./"`:
     ```typescript
     export default defineConfig({
       base: './',
       // ...
     })
     ```
  2. 删除 `handler.go` 中的 `ReplaceAll` 替换逻辑（第 64-68 行）:
     ```go
     // 删除以下代码:
     if pathPrefix != "" {
         result = bytes.ReplaceAll(result, []byte(`"/assets/`), []byte(`"`+pathPrefix+`/assets/`))
         result = bytes.ReplaceAll(result, []byte(`"/favicon`), []byte(`"`+pathPrefix+`/favicon`))
     }
     ```
  3. Vue Router 和 API 的 base path 已通过 `window.__E5_CONFIG__.pathPrefix` 运行时处理，无需额外改动
  4. 验证：以带 pathPrefix 和不带 pathPrefix 两种模式启动，确认静态资源加载正常

---

## 批次 4：低优先级改进

### 4.1 JWT 添加 Issuer 声明
- **文件**: `backend/services/security/jwt.go`
- 添加 `Issuer: "e5-renewal"` 到 Claims，并在 `ParseJWT` 中验证

### 4.2 CI golangci-lint 升级到 v2
- **文件**: `.github/workflows/ci.yml:30`, `backend/.golangci.yml`
- `version: latest` → 固定为 v2 稳定版（如 `version: v2.1.0`）
- 同时需要适配 `.golangci.yml` 配置格式变更（v2 有 breaking changes）

### 4.3 Dependabot 添加 Docker 生态
- **文件**: `.github/dependabot.yml`
- 添加 `package-ecosystem: docker`

### 4.4 login/store.go 使用 repository 层
- **文件**: `backend/services/login/store.go:73, 84`
- 将直接 DB 调用改为通过 `database.Settings.Get()` 和 `database.Settings.Upsert()`

---

## 验证步骤

每批完成后执行:

```bash
# 后端
cd backend && go test -race ./... && golangci-lint run

# 前端
cd frontend && npx vitest run && npx eslint src/

# 全量构建
docker build -t e5-renewal:test .
```

## 关键文件清单

| 文件 | 批次 | 修改类型 |
|------|------|----------|
| `backend/spa/handler.go` | 1, 3 | XSS 修复 + 删除 ReplaceAll hack |
| `backend/main.go` | 1, 3 | 并发修复 + gin.New() |
| `backend/handlers/oauth.go` | 1 | 安全修复（POST + postMessage origin） |
| `frontend/src/api/client.ts` | 1 | 添加 401 拦截器 |
| `frontend/src/components/AccountFormDialog.vue` | 1 | origin 校验 + API 改 POST |
| `backend/services/scheduler/scheduler.go` | 2 | 通知配置双重加载修复 |
| `frontend/src/router/index.ts` | 3 | catch-all + 重定向 |
| `frontend/src/views/DashboardView.vue` | 3 | 响应性修复 |
| `frontend/src/components/AppSidebar.vue` | 3 | 暗色模式监听 |
| `frontend/src/components/ConfirmDialog.vue` | 3 | i18n 默认文本 |
| `frontend/src/i18n/index.ts` | 3 | 添加缺失 key |
| `frontend/vite.config.ts` | 3 | base 改为 `"./"` |
| `backend/database/account.go` | 3 | Pluck 错误检查 |
| `backend/services/graph/caller.go` | 3 | body drain |
| `backend/services/security/jwt.go` | 4 | issuer 声明 |
| `.github/workflows/ci.yml` | 4 | golangci-lint v2 升级 |
| `backend/.golangci.yml` | 4 | 适配 v2 配置格式 |
| `backend/services/login/store.go` | 4 | repository 层 |
