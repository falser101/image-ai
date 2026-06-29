# AI 产品图生成平台

一个可用、数据持久化、前后端分离的最小实现。员工上传产品图，AI 自动提取卖点 + 生图 Prompt，管理员配置模型/风格预设/OSS，生成结果落入图库，支持筛选与下载。

## 目录

- `backend/` Go + Gin + GORM + SQLite + JWT
- `frontend/` Vue 3 + Vite + Element Plus + Pinia
- `backend/data.db` 首次启动自动创建
- `backend/uploads/` 上传文件与生成图

## 快速开始

```bash
# 1) 启动后端（默认 :8080，首次自动初始化 admin / admin123）
cd backend
go run .

# 2) 另一个终端启动前端（默认 :5173，已配置代理 /api → :8080）
cd frontend
npm install
npm run dev
```

浏览器访问 <http://localhost:5173> ，使用 `admin / admin123` 登录。

### 生产构建

```bash
cd frontend && npm run build       # 产物在 frontend/dist
# 用 nginx 或 vite preview 静态托管
```

后端单文件部署：

```bash
cd backend && go build -o image-ai-server .
PORT=8080 ./image-ai-server
```

## 功能映射

| 需求 | 实现位置 |
| --- | --- |
| 多账户数据隔离 | `handlers/*.go` 中所有列表/详情按 `user_id` 过滤，管理员可见全部 |
| 操作日志 | `middleware/logger.go` 自动落库，UI 在 `views/OperationLogs.vue` |
| 账户管理（增删改查） | `views/Users.vue` + `handlers/user.go` |
| 卖点录入与历史 | 上传后自动写入 + `views/SellingPoints.vue` + `views/Products.vue` 内嵌抽屉 |
| AI 解析/生图 | `services/ai.go`（支持 OpenAI 兼容协议，无配置自动 mock） |
| 模型选择 | `views/ModelConfigs.vue` + `views/Upload.vue` 下拉 |
| 图库筛选/下载 | `views/Gallery.vue` + `GET /api/gallery/:id/file`（下载即 `target=_blank`） |
| 平台配置 | `views/ModelConfigs.vue`、`views/OssConfig.vue`、`views/StylePresets.vue` |

## AI 模型说明

- 在 **模型配置** 页面新增模型，**三步配好**：选 Provider → 填 API Key → 选模型（预置常用模型，也可输入自定义）。
- 协议：与 OpenAI 兼容的 `chat/completions` 和 `images/generations`。国内可填 DashScope/DeepSeek/智谱/通义等网关。
- **MiniMax（海螺 image-01 / image-02 / image-01-live）** 已原生支持：
  - 自动按 `POST {base}/v1/image_generation` 协议调用
  - body = `{model, prompt, aspect_ratio, response_format:"url", n:1, prompt_optimizer:true}`
  - **subject_reference**：如果生成请求带 `sourceImageId`（即在上传与生图页先上传了一张原图），后端会自动把它作为 `subject_reference: [{type, image_file}]` 附在请求里；`type` 根据文件名自动判 `product` 或 `character`，`image_file` 用 `data:image/...;base64,...` 内嵌，避免外网拉不到本机图片。
  - aspect_ratio 由前端所选尺寸自动推断（1:1 / 4:3 / 3:4 / 16:9 / 9:16 / 21:9 / 3:2）
- 留空 API Key 或选 `mock` 走内置占位逻辑，保证流程始终可演示。
- **内置 provider 自动锁定 BaseURL / 名称**：`minimax / openai / dashscope / stability / mock` 是内置 provider，UI 上 BaseURL 框只读（不可改），名称会按「`{provider 标签} {模型名}`」自动生成（可手动改）。任何途径（UI / API PUT）试图把内置 provider 的 BaseURL 改成非官方地址都会被强制还原，保证不会因为手抖把请求发到第三方站点。需要自定义地址请选 `自定义（OpenAI 兼容）` 这一项。
- **自动降级**：当配置的视觉/生图接口调用失败（404、网络超时、鉴权失败等任意错误）时，系统自动生成 SVG 占位图，gallery 记录状态为 `fallback`，UI 显示「占位图」徽标和降级原因；不会因模型未配好而阻断业务流程。
- **OSS 配置** 暂以元信息保存（endpoint/key/secret 等），启用后用于将来切换远端存储；当前实现保留本地存储，文件保存到 `backend/uploads/`，由 `GET /uploads/...` 静态访问。

## 默认账号

| 账号 | 密码 | 角色 |
| --- | --- | --- |
| admin | admin123 | 管理员 |

环境变量可覆盖：

- `PORT` `DB_PATH` `UPLOAD_DIR` `JWT_SECRET`
- `ADMIN_USERNAME` `ADMIN_PASSWORD`

## API 速查

| Method | Path | 说明 |
| --- | --- | --- |
| POST | /api/auth/login | 登录获取 token |
| POST | /api/auth/register | 员工自助注册 |
| GET  | /api/auth/me | 当前用户 |
| POST | /api/ai/analyze | 上传图片（multipart）→ 卖点 + Prompt + 自动建产品 |
| POST | /api/ai/generate | 用 prompt + 模型 + 风格 生成图片 |
| GET  | /api/ai/tasks/:id | 任务状态 |
| GET  | /api/products | 产品列表（员工只看自己） |
| GET  | /api/gallery | 图库（支持 modelConfigId/styleId/keyword 筛选） |
| GET  | /api/selling-points | 卖点历史 |
| GET  | /api/model-configs | 模型配置（管理员） |
| GET  | /api/style-presets | 风格预设 |
| GET  | /api/users | 员工列表（管理员） |
| GET  | /api/operation-logs | 操作日志（管理员） |

## 端到端冒烟测试

```bash
# 后端
cd backend && go build -o /tmp/server . && /tmp/server &
TOKEN=$(curl -s -X POST localhost:8080/api/auth/login -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' | python3 -c 'import json,sys;print(json.load(sys.stdin)["data"]["token"])')
curl -s localhost:8080/api/auth/me -H "Authorization: Bearer $TOKEN"
```
