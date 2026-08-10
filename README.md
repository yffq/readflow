# Readflow

A self-hosted read-later service. Save web articles and paste full-text content for distraction-free reading. Supports Chrome extension, Web UI, and WeChat Mini Program.

自托管的稍后阅读（read-later）工具，支持保存网页文章和粘贴全文内容。支持 Chrome 浏览器扩展、网页端和微信小程序。

## 部署

```bash
git clone https://github.com/yffq/readflow.git
cd readflow

# 1. 修改 Caddyfile 中的域名为你自己的域名
# 2. 启动
docker compose -f docker-compose.prod.yml up -d
```

首次访问后需要创建密码（Setup 页面）。

## API

所有 API 端点都需要 API Key 认证。在 Settings 页面生成 API Key。

### 认证方式

请求头携带 API Key（二选一）：

```bash
# Header（推荐）
Authorization: Bearer rf_xxx...

# URL 参数
?api_key=rf_xxx...
```

### 端点

#### 保存文章 `POST /api/v1/save`

支持传入 URL 或 HTML 全文。

```bash
# 方式一：传入 URL（自动抓取内容）
curl -X POST https://your-domain.com/api/v1/save \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/article"}'

# 方式二：传入 HTML 全文
curl -X POST https://your-domain.com/api/v1/save \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"title":"My Article","html":"<p>Full content...</p>"}'

# 方式三：HTML + URL
curl -X POST https://your-domain.com/api/v1/save \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/article","html":"<p>Content...</p>","title":"Title"}'
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `url` | string | url/html 二选一 | 文章链接 |
| `html` | string | url/html 二选一 | 文章全文 HTML |
| `title` | string | 否 | 文章标题 |

响应：
```json
{"id":"abc123...","title":"Article Title","status":"created"}
```

#### 导出文章 `GET /api/v1/export`

```bash
curl "https://your-domain.com/api/v1/export?limit=50" \
  -H "Authorization: Bearer YOUR_API_KEY"

# 增量导出（只获取某时间后更新的文章）
curl "https://your-domain.com/api/v1/export?updated_after=2024-01-01T00:00:00Z&limit=100" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `limit` | int | 100 | 每页数量 |
| `offset` | int | 0 | 偏移量 |
| `updated_after` | string | - | RFC3339 时间，只返回此时间后更新的文章 |

响应：
```json
{
  "count": 42,
  "next": "/api/v1/export?limit=100&offset=100",
  "results": [
    {
      "id": "abc123...",
      "title": "Article Title",
      "url": "https://example.com/article",
      "author": "Author Name",
      "site_name": "Example",
      "content_html": "<p>...</p>",
      "content_markdown": "...",
      "word_count": 1234,
      "source": "url",
      "extraction_failed": false,
      "status": "unread",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### 批量删除 `POST /api/v1/delete`

```bash
curl -X POST https://your-domain.com/api/v1/delete \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"ids":["article-id-1","article-id-2"]}'
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `ids` | string[] | 是 | 要删除的文章 ID 列表 |

响应：
```json
{"deleted":2}
```

#### Inoreader Automation Webhook

在 Inoreader 的 Automation Rule 中添加 **Trigger webhook**，Webhook URL 填写：

```text
https://your-domain.com/api/v1/webhooks/inoreader/YOUR_API_KEY
```

建议规则使用“文章被添加指定标签”作为触发条件。Readflow 会直接保存 Inoreader
发送的 HTML 正文，并保留标题、作者、Feed 名称和原文链接。重复的原文链接不会再次保存。

其中 `YOUR_API_KEY` 是完整的 `rf_...` Key，不要添加 `Bearer`。Webhook URL 包含
API Key，只能通过 HTTPS 使用。建议为 Inoreader 单独创建一个 API Key，
便于独立撤销和轮换。

## 浏览器扩展

在 Chrome 中加载 `extension/` 目录即可使用：

1. 打开 `chrome://extensions/`
2. 开启「开发者模式」
3. 点击「加载已解压的扩展程序」，选择 `extension/` 目录
4. 在扩展图标上右键 → 选项，配置 Server URL 和 API Key

**使用方式：**
- 点击扩展图标 → Save Current Page 保存当前页面
- 页面右键 → Save Page to Readflow
- 链接右键 → Save Link to Readflow

**剪藏到 Obsidian：**

无需安装任何 Obsidian 插件，使用原生 `obsidian://` URI 协议。

1. 右键扩展图标 → 选项，开启 Obsidian Clipping 并配置：
   - **Vault name**：可选，不填则写入当前活跃仓库
   - **Folder paths**：支持多个目录（如 Readflow、Work、Personal），可增删。上次使用的目录高亮显示
2. 确保 Obsidian 正在运行
3. 在 readflow 文章详情页，右下角出现所有目录的剪藏按钮，点击任一即可保存到对应目录
4. 也可在扩展弹窗中点击对应目录按钮，上次使用的目录高亮显示

笔记以 Markdown 格式保存，文件名格式为 `{日期} {标题}.md`，并包含 YAML frontmatter 元数据（标题、作者、来源、标签等）。

极少数超长文章（URI 超过 1.5MB）会自动创建含元数据的空白笔记，正文复制到剪贴板，粘贴到笔记中即可。

## 微信小程序

在微信开发者工具中打开 `miniprogram/` 目录即可运行。

1. 首次启动会自动跳转设置页，配置服务器地址和 API Key
2. 设置页保存后进入文章列表，支持下拉刷新和删除
3. 点击文章进入阅读视图，自动适配手机屏幕
4. 点击原文链接会将 URL 复制到剪贴板

## 开发

```bash
# 启动开发服务器
go run ./cmd/server
# 访问 http://localhost:8080

# 运行所有测试
go test ./...

# 运行扩展单元测试
node extension/test.js
node extension/save-test.js
node extension/obsidian-navigation-test.js
```
