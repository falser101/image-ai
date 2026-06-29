# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概览

AI 产品图生成平台：员工上传产品图 → AI 视觉模型提取卖点 + 生图 prompt → 调用生图 API 出图 → 落库图库。前后端分离，后端 Go + SQLite，前端 Vue 3 + Element Plus。

## 常用命令

```bash
# 后端（默认 :8080，监听所有接口）
cd backend
go build -o /tmp/server . && /tmp/server      # 构建并跑
go build ./...                                  # 仅构建检查
go run .                                        # 直接跑（开发用）

# 前端（默认 :5173，proxy /api 和 /uploads → :8080）
cd frontend
npm install
npm run dev          # 开发
npm run build        # 生产构建到 dist/

# 端到端冒烟
TOKEN=$(curl -s -X POST localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c 'import json,sys;print(json.load(sys.stdin)["data"]["token"])')
curl -s localhost:8080/api/auth/me -H "Authorization: Bearer $TOKEN"
```

重启后端时旧进程经常占着 8080 不放：
```bash
ss -tlnp | grep 8080
kill -9 <pid>          # 必须 -9，Go 默认忽略 SIGTERM
```

## 数据落盘位置

| 文件 | 内容 |
|---|---|
| `backend/data.db` | SQLite，元数据（用户/产品/卖点/原图/生成图/模型配置/操作日志/AI 任务） |
| `backend/uploads/` | 上传原图 + 生成图（`r.Static("/uploads", cfg.UploadDir)` 暴露） |

迁移/备份要整块搬。环境变量 `DB_PATH` `UPLOAD_DIR` `PORT` `JWT_SECRET` `ADMIN_USERNAME` `ADMIN_PASSWORD` 可覆盖。

## 后端架构

**职责分层**：
- `models/models.go` — GORM 表结构，所有敏感字段用 `json:"-"`（如 `User.Password`）
- `handlers/` — HTTP 入口（每个领域一个文件：auth / user / product / ai / image / gallery / configs / selling_point / operation_log / provider_catalog）
- `services/ai.go` — AI 调用业务逻辑（OpenAI 兼容 + MiniMax 私有协议双协议）
- `middleware/` — JWT 鉴权 + 自动落操作日志
- `utils/` — JWT、bcrypt、统一响应 `{code, data, msg}`

**路由分组**（`main.go`）：`/api/auth/*` 公开 → `authed` 走 JWT 中间件 → `admin` 子组走 `RequireAdmin()`。`/api/providers` 和 `/api/health` 是公开的（前端下拉 provider 时不带 token）。

**关键约定**：

1. **API Key 永远不在响应里回显明文**。`handlers/configs.go` 的 `maskAPIKeys()` 把所有 `ModelConfig` 响应的 `apiKey` 字段替换成 `"****"`（有值）或 `""`（空）。前端 `ModelConfigs.vue` 的 `openEdit` 也强制 `apiKey: ''` 不预填。
2. **Update 留空 = 不改**。前端 `submit` 里 `if (!data.apiKey) delete data.apiKey`；后端 `Update` handler 检测到 `patch.APIKey == ""` 时把原值赋回 patch，结构体走 `Updates(patch)`（不用 map，GORM 不会因零值覆盖）。
3. **内置 provider 锁 URL**。`providerCatalog()` 返回的 `minimax` 是 `builtIn=true`；`Create` 自动补 `baseUrl`、自动按 `{label} {model}` 命名；`Update` 拒绝任何非官方 `baseUrl`，会被强制还原。`provider_catalog.go` 的 `List` 是前端下拉源。
4. **AI 调用一律 graceful fallback**。`services/ai.go:Generate()` 任何错误（404、超时、401、`base_resp.status_code != 0`）都返回 `mockGenerate` 生成的 SVG 占位，gallery `Status="fallback"`，前端显示「占位图」徽标。**不要让 AI 配置错误把业务流程掐断**。
5. **MiniMax 私有协议**。`callMinimaxImage` 走 `POST {base}/v1/image_generation`，body = `{model, prompt, aspect_ratio, response_format:"url", n, prompt_optimizer?}`。**`subject_reference.type` 只接受 `"character"`**（商品会被 MiniMax 拒绝 2013）。`buildSubjectReference` 固定返回 character，并把图读成 `data:image/...;base64,...` 内嵌（避免外网拉不到内网图）。`UseAsSubject` bool 控制是否附加 subject_reference（前端 Upload.vue 勾选框）。
6. **MiniMax 业务错误要看 `base_resp.status_code`**。HTTP 200 但 status_code != 0 是业务错误，要 surface `status_msg` 给用户，不能只看 HTTP 状态或 `image_urls` 是否存在。
7. **`pickAspectRatio`** 把 W×H 距离最小匹配到 8 个合法值（1:1/4:3/3:4/16:9/9:16/21:9/3:2/2:3），加新比例时同步改 `provider_catalog.go` 的 Description。
8. **mock 是 fallback，不是 provider**。`provider_catalog.go` 不再有 `mock` 条目；运行时仅当 `cfg.APIKey == ""` 时走 `mockAnalyze`/`mockGenerate`，确保流程可演示。

## 前端架构

- `src/api/index.js` 集中所有后端调用，**改 API 路径先改这里**。`http.js` 是 axios 封装，token 拦截器在里头。
- `src/views/` 一个页面一个文件，对应侧边栏菜单。`MainLayout.vue` 包侧边栏 + topbar + `<router-view>`。
- `Upload.vue` 是主流程入口，分两步：① 上传图 + AI 解析（卖点 + prompt）→ ② 选模型风格 + 生图。`useAsSubject` 勾上才会把原图作为 `subject_reference`。
- `ModelConfigs.vue` 三步 wizard 弹窗：Provider/类型 → API Key → 选模型。内置 provider 的 BaseURL 是 readonly，name 自动填。**编辑模式 apiKey 输入框永远空，留空 = 不修改**。
- `Gallery.vue` 显示结果，`status="fallback"` 时显示「占位图」badge 和降级原因。
- Vite dev server proxy `/api`、`/uploads` → `:8080`，所以开发时前端不需要配 baseURL。
- 默认 admin / admin123（首次启动自动 seed）。

## 调试备忘

- 看后端进程占的端口：`ss -tlnp | grep 8080`（`lsof -i` 也行）
- SQLite 直接看：`sqlite3 backend/data.db ".tables"` 然后 `.schema galleries`
- 上传文件直接看：`ls backend/uploads/`
- 前端改了立即生效（Vite HMR），后端改了必须 `go build` + 重启
- 前端 dist 在 `frontend/dist/`，Vite 默认配置 `outDir: 'dist'` `emptyOutDir: true`
- 端到端验证一个完整生图流：登录 → `POST /api/ai/analyze`（multipart）→ 拿 `imageId` → `POST /api/ai/generate` 带 `sourceImageId` + `useAsSubject: true`
- 验证 API Key 防护：直接 `GET /api/model-configs` 看到 `apiKey: "****"` 才对
- 验证内置锁：`PUT /api/model-configs/4 -d '{"baseUrl":"https://evil.com"}'` 后再 GET，应还原成 `https://api.minimaxi.com`

## 已知边界 / 不做的事

- OSS 配置：只存元信息，**不实际切换存储后端**——文件永远落本地 `uploads/`
- vision 模型配置：没配 vision 模型时，`Analyze` 走 `mockAnalyze`（硬编码 6 条中文卖点 + 固定英文 prompt）。要让真模型分析，管理员需在「模型配置」添加 `type=vision` 的记录
- 前端不做精细权限控制：员工/管理员路由在 router 里没拦，后端 `RequireAdmin()` 是真实防线
- 没用 rate limit、没有重试、没有 worker pool；MiniMax 一张图就发一次 HTTP
