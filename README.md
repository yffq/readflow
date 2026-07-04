# Readflow

自托管的稍后阅读（read-later）工具，支持保存网页文章和粘贴全文内容。

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

## 微信小程序

在微信开发者工具中打开 `miniprogram/` 目录即可运行。

1. 首次启动会自动跳转设置页，配置服务器地址和 API Key
2. 设置页保存后进入文章列表，支持下拉刷新和删除
3. 点击文章进入阅读视图，自动适配手机屏幕
4. 点击原文链接会将 URL 复制到剪贴板

## 开发

```bash
go run ./cmd/server
# 访问 http://localhost:8080
```
